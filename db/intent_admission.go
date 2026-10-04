package db

// 한국어 읽기 안내
// 추가 intent를 만들었으나 작업 실행 허가 단계가 실패한 경우 저장 결과를 보상하는 작은 트랜잭션이다.
// 호출자가 작업 실행 gate를 계속 보유해야 삭제 전에 다른 worker가 intent를 선점하지 않는다. DB의 open 조건만으로 호출 측 gate를 대체하지 않는다.
// activity는 노드 삭제 시 NULL 참조로 남을 수 있으므로 먼저 지우고, open intent 한 행이 실제 제거되었는지 확인한다.

import "fmt"

// DiscardOpenIntent compensates a follow-up creation when task admission fails.
// The task execution gate must still be held by the caller, so the intent cannot
// be claimed between this check and deletion. Edges and anchors cascade with the
// node; activity is deleted explicitly because its node FK otherwise becomes NULL.
// 한국어: 실행 허가 실패를 보상한다. 호출 측 gate를 계속 잡은 상태에서 activity를 지우고 아직 open인 한 intent만 삭제해 선점된 작업의 결과를 지우지 않도록 한다.
func (s *ExplorationStore) DiscardOpenIntent(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM activity WHERE exploration_id=$1 AND node_id=$2`, s.expID, id); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM exploration_nodes
		WHERE id=$1 AND exploration_id=$2 AND kind='intent' AND state='open'`, id, s.expID)
	if err != nil {
		return err
	}
	removed, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if removed != 1 {
		return fmt.Errorf("open intent %d was not available for admission rollback", id)
	}
	return tx.Commit()
}
