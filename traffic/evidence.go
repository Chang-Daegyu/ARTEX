// [한국어 파일 안내] traffic/evidence.go
// 수집된 트래픽에서 생략되지 않은 본문을 evidence 저장소로 전달하는 연결부다.
// 일반 Get은 미리보기를 반환할 수 있으므로 증거 복사는 별도의 Reader 경로를 사용한다.
// 쓰기 잠금을 잡은 콜백 안에서 파일을 읽고 닫아 원시 트래픽 삭제와 복사가 엇갈리지 않게 한다.
package traffic

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// EvidenceExchange exposes complete captured bodies only during the callback.
// Readers are closed before releasing the writer lock, so host deletion cannot
// remove a blob between its index lookup and its evidence copy.
// 한국어 자료형: 증거 복사 중에만 유효한 원문 Reader와 길이·해시·헤더를 묶는다. Request/Response를 장기 보관할 수 있는 DTO가 아니다.
type EvidenceExchange struct {
	ID, URL, Method, ContentType string
	TS                           int64
	Status                       int
	ReqHead, RespHead            string
	ReqLen, RespLen              int64
	ReqHash, RespHash            string
	Request, Response            io.Reader
}

// 한국어 해설: 요청한 모든 ID를 쓰기 잠금 아래 차례로 읽어 consume에 전달한다. 콜백 밖으로 Reader를 보관하면 수명 계약을 어기게 된다.
func (t *Traffic) ReadEvidence(ctx context.Context, ids []string, consume func(EvidenceExchange) error) error {
	if t == nil {
		return errors.New("流量录制存储不可用")
	}
	t.wmu.Lock()
	defer t.wmu.Unlock()
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := t.readEvidence(id, consume); err != nil {
			return fmt.Errorf("流量 %s: %w", id, err)
		}
	}
	return nil
}

// 한국어 해설: SQLite 인라인 본문, 외부 blob, 과거 .http 파일을 공통 EvidenceExchange로 바꾼다. 외부 파일은 콜백 종료 전에 닫힌다.
func (t *Traffic) readEvidence(id string, consume func(EvidenceExchange) error) error {
	e := EvidenceExchange{ID: id}
	var legacy string
	if err := t.db.QueryRow(`SELECT ts,url,method,status,content_type,req_len,resp_len,path FROM exchanges WHERE id=?`, id).
		Scan(&e.TS, &e.URL, &e.Method, &e.Status, &e.ContentType, &e.ReqLen, &e.RespLen, &legacy); err != nil {
		return err
	}
	var req, resp []byte
	var reqBlob, respBlob sql.NullString
	err := t.db.QueryRow(`SELECT req_head,req_body,req_blob,resp_head,resp_body,resp_blob FROM exchange_bodies WHERE id=?`, id).
		Scan(&e.ReqHead, &req, &reqBlob, &e.RespHead, &resp, &respBlob)
	if errors.Is(err, sql.ErrNoRows) && legacy != "" {
		if !filepath.IsLocal(legacy) {
			return errors.New("旧流量路径无效")
		}
		r, err := os.Open(filepath.Join(t.dir, legacy, "request.http"))
		if err != nil {
			return err
		}
		defer r.Close()
		p, err := os.Open(filepath.Join(t.dir, legacy, "response.http"))
		if err != nil {
			return err
		}
		defer p.Close()
		e.ReqHead, e.Request, err = splitLegacyEvidence(r)
		if err != nil {
			return err
		}
		e.RespHead, e.Response, err = splitLegacyEvidence(p)
		if err != nil {
			return err
		}
		return consume(e)
	}
	if err != nil {
		return err
	}
	e.Request = bytes.NewReader(req)
	e.Response = bytes.NewReader(resp)
	if reqBlob.Valid && reqBlob.String != "" {
		f, _, err := t.Blob(reqBlob.String)
		if err != nil {
			return err
		}
		defer f.Close()
		e.Request = f
		e.ReqHash = reqBlob.String
	}
	if respBlob.Valid && respBlob.String != "" {
		f, _, err := t.Blob(respBlob.String)
		if err != nil {
			return err
		}
		defer f.Close()
		e.Response = f
		e.RespHash = respBlob.String
	}
	return consume(e)
}

// 한국어 해설: 과거 HTTP 파일에서 빈 줄까지 헤더를 읽고 나머지를 본문 Reader로 돌려준다. 헤더가 1 MiB를 넘으면 거절한다.
func splitLegacyEvidence(r io.Reader) (string, io.Reader, error) {
	b := bufio.NewReader(r)
	var head strings.Builder
	for {
		line, err := b.ReadString('\n')
		if line == "\n" || line == "\r\n" {
			return head.String(), b, nil
		}
		head.WriteString(line)
		if head.Len() > 1<<20 {
			return "", nil, errors.New("旧流量报文头过大")
		}
		if errors.Is(err, io.EOF) {
			return head.String(), b, nil
		}
		if err != nil {
			return "", nil, err
		}
	}
}
