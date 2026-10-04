package db

// 한국어 읽기 안내
// 탐색 그래프의 오래된 부분을 요약하는 compactor가 사용하는 영속 상태를 담당한다. 요약문 생성 자체는 agent 쪽에서 수행한다.
// round_no는 해당 작업의 planner 실행 회차, cold_since_round는 마지막으로 차가워진 회차, content_version은 요약의 원본 변경 감지 값이다.
// digest 노드와 covers 간선을 함께 기록한다. 요약이 교체되어도 원래 fact/intent와 이전 digest 노드는 상세 추적을 위해 남는다.
// 활성 digest의 covers 관계가 현재 접힌 원본의 기준이며 SupersedeDigests는 활성 상태와 covers 제거를 한 트랜잭션으로 맞춘다.

// cold-digest §1/§2.3/§5: persistence for cold-node compression — the round
// counter, cold_since_round stamps, content versions, digest nodes and the
// covers edges that are the source of truth for "which digest folds node X".

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// BumpRound advances this exploration's planner-round counter and returns the
// new value (§2.3). Called once per planner wake-up.
// 한국어: planner가 깨어나 처리한 한 회차마다 해당 exploration의 round_no를 원자적으로 증가시킨다. 전역 노드 ID나 실제 경과 초를 냉각 기준으로 쓰지 않는다.
func (s *ExplorationStore) BumpRound() (int64, error) {
	var r int64
	err := s.db.QueryRow(`UPDATE explorations SET round_no = round_no + 1 WHERE id=$1 RETURNING round_no`, s.expID).Scan(&r)
	return r, err
}

// RoundNo returns the current planner-round counter.
func (s *ExplorationStore) RoundNo() (int64, error) {
	var r int64
	err := s.db.QueryRow(`SELECT round_no FROM explorations WHERE id=$1`, s.expID).Scan(&r)
	return r, err
}

// ColdStamps returns cold_since_round for every foldable node (intent/fact):
// id → *round (nil when the node is hot / unstamped). Used to compute the ≥R
// debounce and to know which stamps to set/clear this round.
// 한국어: 접을 수 있는 intent/fact의 냉각 시작 회차를 읽는다. nil은 아직 차가운 상태로 찍히지 않은 노드로 해석한다.
func (s *ExplorationStore) ColdStamps() (map[int64]*int64, error) {
	rows, err := s.db.Query(`SELECT id, cold_since_round FROM exploration_nodes
		WHERE exploration_id=$1 AND kind IN ('intent','fact')`, s.expID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]*int64{}
	for rows.Next() {
		var id int64
		var cs sql.NullInt64
		if err := rows.Scan(&id, &cs); err != nil {
			return nil, err
		}
		if cs.Valid {
			v := cs.Int64
			out[id] = &v
		} else {
			out[id] = nil
		}
	}
	return out, rows.Err()
}

// StampOp is one cold_since_round change: Set stamps Round; !Set clears it.
type StampOp struct {
	ID    int64
	Set   bool
	Round int64
}

// ApplyStampOps writes a batch of cold_since_round changes in one transaction.
// 한국어: 여러 노드의 냉각 시작/해제를 한 트랜잭션에 반영한다. compactor가 계산한 상태를 저장하며 무엇이 hot인지 자체 판단하지는 않는다.
func (s *ExplorationStore) ApplyStampOps(ops []StampOp) error {
	if len(ops) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, o := range ops {
		if o.Set {
			if _, err := tx.Exec(`UPDATE exploration_nodes SET cold_since_round=$1 WHERE id=$2 AND exploration_id=$3`, o.Round, o.ID, s.expID); err != nil {
				return err
			}
		} else {
			if _, err := tx.Exec(`UPDATE exploration_nodes SET cold_since_round=NULL WHERE id=$1 AND exploration_id=$2`, o.ID, s.expID); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// ContentVersions returns id → content_version for all nodes (§5.3 signature).
// 한국어: 원본 변경 여부를 비교할 id→버전 맵을 반환한다. 요약을 재사용할 때 같은 노드 ID라는 이유만으로 최신 본문이라고 판단하지 않도록 한다.
func (s *ExplorationStore) ContentVersions() (map[int64]int, error) {
	rows, err := s.db.Query(`SELECT id, content_version FROM exploration_nodes WHERE exploration_id=$1`, s.expID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]int{}
	for rows.Next() {
		var id int64
		var v int
		if err := rows.Scan(&id, &v); err != nil {
			return nil, err
		}
		out[id] = v
	}
	return out, rows.Err()
}

// ActiveDigests returns the exploration's live digest nodes (state='active'),
// oldest first.
func (s *ExplorationStore) ActiveDigests() ([]*Node, error) {
	rows, err := s.db.Query(`SELECT `+nodeCols+` FROM exploration_nodes
		WHERE exploration_id=$1 AND kind=$2 AND state=$3 ORDER BY id`, s.expID, KindDigest, StateDigestActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNodes(rows)
}

// AddDigest writes one digest node and its covers edges (digest→member) in a
// single transaction. payload is the digest body + member_ids + generation +
// signature (see cold-digest §1). Returns the new digest id.
// 한국어: 요약 노드와 digest→원본 covers 간선을 함께 저장한다. 중간 오류가 나면 둘 다 롤백되어 연결 없는 요약이 남지 않는다.
func (s *ExplorationStore) AddDigest(payload map[string]any, memberIDs []int64) (int64, error) {
	raw, _ := json.Marshal(payload)
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var id int64
	if err := tx.QueryRow(`
INSERT INTO exploration_nodes(exploration_id, kind, payload, priority, state, origin)
VALUES ($1, $2, $3, 0, $4, 'compactor') RETURNING id`,
		s.expID, KindDigest, string(raw), StateDigestActive).Scan(&id); err != nil {
		return 0, err
	}
	// 한국어: covers는 어떤 원본이 이 요약에 접혔는지 나타낸다. 원본 노드 자체를 지우거나 덮어쓰지 않는다.
	for _, m := range memberIDs {
		if m == id {
			continue
		}
		if _, err := tx.Exec(`
INSERT INTO exploration_edges(exploration_id, src_id, rel, dst_id) VALUES ($1,$2,$3,$4)
ON CONFLICT (exploration_id, src_id, rel, dst_id) DO NOTHING`, s.expID, id, RelCovers, m); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}

// CoveredMembers maps member id → covering digest id, for ACTIVE digests only
// (§6.1 point 1). A node with no entry is not currently folded. Should a member
// carry covers edges from two digests (a torn major write), the lowest digest id
// wins deterministically — callers dedupe on this.
// 한국어: 활성 digest만 골라 각 원본을 접는 digest ID를 구한다. 동일 원본을 여러 활성 digest가 가리키면 가장 작은 ID를 택해 결과를 결정적으로 만든다.
func (s *ExplorationStore) CoveredMembers() (map[int64]int64, error) {
	rows, err := s.db.Query(`
SELECT e.dst_id, e.src_id
FROM exploration_edges e
JOIN exploration_nodes d ON d.id=e.src_id AND d.exploration_id=e.exploration_id
WHERE e.exploration_id=$1 AND e.rel=$2 AND d.kind=$3 AND d.state=$4
ORDER BY e.src_id`, s.expID, RelCovers, KindDigest, StateDigestActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]int64{}
	for rows.Next() {
		var member, digest int64
		if err := rows.Scan(&member, &digest); err != nil {
			return nil, err
		}
		if _, seen := out[member]; !seen { // first (lowest digest id) wins
			out[member] = digest
		}
	}
	return out, rows.Err()
}

// DigestMembers returns the member ids a digest covers (covers edges), sorted.
func (s *ExplorationStore) DigestMembers(digestID int64) ([]int64, error) {
	rows, err := s.db.Query(`SELECT dst_id FROM exploration_edges
		WHERE exploration_id=$1 AND src_id=$2 AND rel=$3`, s.expID, digestID, RelCovers)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, rows.Err()
}

// SupersedeDigests retires digest nodes (state→superseded) AND removes their
// covers edges, atomically, so the active-coverage set (CoveredMembers) never
// double-counts a member during a major recompaction (§5.1). The digest node
// itself is kept (node_detail can still resolve it).
// 한국어: 이전 요약의 covers를 없애고 상태를 superseded로 바꾼다. 요약 노드는 유지하므로 예전 링크로 내용 확인은 가능하다.
func (s *ExplorationStore) SupersedeDigests(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// 한국어: 연결 제거와 비활성 상태 전환을 함께 커밋해 활성 요약의 원본 집계가 엇갈리지 않게 한다.
	for _, id := range ids {
		if _, err := tx.Exec(`DELETE FROM exploration_edges
			WHERE exploration_id=$1 AND src_id=$2 AND rel=$3`, s.expID, id, RelCovers); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE exploration_nodes SET state=$1 WHERE id=$2 AND exploration_id=$3 AND kind=$4`,
			StateDigestSuperseded, id, s.expID, KindDigest); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// NodeAssets maps each of the given node ids → the asset ids it is anchored to
// (exploration_anchors). Used to group cold digests by asset (§6.2 index).
// 한국어: 선택한 탐색 노드의 전역 자산 앵커를 읽어 요약을 자산별로 묶는 데 사용한다. 자산 자체의 계층 관계를 만드는 함수는 아니다.
func (s *ExplorationStore) NodeAssets(ids []int64) (map[int64][]int64, error) {
	out := map[int64][]int64{}
	if len(ids) == 0 {
		return out, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, s.expID)
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}
	rows, err := s.db.Query(`SELECT a.node_id, a.asset_id
		FROM exploration_anchors a
		JOIN exploration_nodes n ON n.id=a.node_id
		WHERE n.exploration_id=$1 AND a.node_id IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var node, asset int64
		if err := rows.Scan(&node, &asset); err != nil {
			return nil, err
		}
		out[node] = append(out[node], asset)
	}
	return out, rows.Err()
}
