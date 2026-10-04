package db

// 한국어 읽기 안내
// 현재 작업이 직접 연결한 source 작업의 기록을 읽기 전용 문맥으로 합치는 조회 계층이다. source의 source까지 재귀로 확대하지 않는다.
// DirectSourceStores는 매번 살아 있는 관계를 조회하므로 삭제된 작업의 오래된 관계를 캐시에 남기지 않는다.
// 상속 노드는 SourceTaskID/Inherited를 붙이고, 상속된 worker 활동은 종료 상태 intent에 속한 기록으로 제한한다. 원본 planner/main 대화 전체를 넘기지 않는다.
// 페이지 조회는 전역적으로 유일한 노드 ID를 기준으로 각 소스의 후보를 병합한다. 파일의 함수는 원본 행을 복제하거나 수정하지 않는다.

import (
	"database/sql"
	"sort"
)

// DirectSourceStore binds one directly related task to its exploration store.
// It is intentionally a read-side helper: callers keep using the receiver store
// for every graph mutation, frontier lookup, and intent claim.
type DirectSourceStore struct {
	Task  TaskSource
	Store *ExplorationStore
}

// DirectSourceStores resolves the live, direct task relations for this
// exploration. It deliberately queries on every call: relations disappear when
// a source task is deleted, and inherited context must not retain stale rows.
// Sources of a source are never expanded.
// 한국어: 현재 exploration에 연결된 살아 있는 직접 source만 매번 조회한다. 관계 변경/삭제가 즉시 반영되며 source의 source까지 펼치지 않는다.
func (s *ExplorationStore) DirectSourceStores() ([]DirectSourceStore, error) {
	rows, err := s.db.Query(`
SELECT source.id, source.exploration_id, source.description, source.goal, source.status
FROM tasks owner
JOIN task_relations relation ON relation.task_id=owner.id
JOIN tasks source ON source.id=relation.source_task_id AND source.deleted_at IS NULL
WHERE owner.exploration_id=$1 AND owner.deleted_at IS NULL
ORDER BY relation.created_at, source.id`, s.expID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DirectSourceStore{}
	for rows.Next() {
		var source TaskSource
		if err := rows.Scan(&source.TaskID, &source.ExplorationID, &source.Description, &source.Goal, &source.Status); err != nil {
			return nil, err
		}
		out = append(out, DirectSourceStore{Task: source, Store: s.db.Exploration(source.ExplorationID)})
	}
	return out, rows.Err()
}

// TaskID returns the live task bound to this exploration. Explorations created
// directly in tests or maintenance code have no task and return zero.
func (s *ExplorationStore) TaskID() (int64, error) {
	var id int64
	err := s.db.QueryRow(`SELECT id FROM tasks WHERE exploration_id=$1 AND deleted_at IS NULL`, s.expID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return id, nil
}

func markInheritedNode(n *Node, taskID int64) *Node {
	if n == nil {
		return nil
	}
	n.SourceTaskID = taskID
	n.Inherited = true
	return n
}

func markInheritedActivities(in []Activity, taskID int64) []Activity {
	for i := range in {
		in[i].SourceTaskID = taskID
		in[i].Inherited = true
	}
	return in
}

func inheritedIntentTerminal(state string) bool {
	switch state {
	case "done", "blocked", "exhausted", "stopped":
		return true
	default:
		return false
	}
}

// ListByKindWithSources returns local nodes followed by nodes from each direct
// source in relation order. The limit remains per exploration, matching the
// existing ListByKind contract while ensuring one large task cannot hide all
// inherited context from another source.
// 한국어: 현재 탐색 뒤에 source 순서대로 노드를 합치며 limit는 각 exploration별로 적용한다. 전체 결과 상한이 항상 limit라고 해석하면 안 된다.
func (s *ExplorationStore) ListByKindWithSources(kind string, limit int) ([]*Node, error) {
	out, err := s.ListByKind(kind, limit)
	if err != nil {
		return nil, err
	}
	sources, err := s.DirectSourceStores()
	if err != nil {
		return nil, err
	}
	for _, source := range sources {
		nodes, err := source.Store.ListByKind(kind, limit)
		if err != nil {
			return nil, err
		}
		for _, node := range nodes {
			if kind == KindIntent && !inheritedIntentTerminal(node.State) {
				continue
			}
			out = append(out, markInheritedNode(node, source.Task.TaskID))
		}
	}
	return out, nil
}

// ListByKindPageWithSources is the paginated, keyword-filterable sibling of
// ListByKindWithSources: it returns one newest-first page (id < before, before<=0
// = newest) spanning this exploration and its direct sources, plus hasMore and the
// filtered total across all of them. Node ids are globally unique, so merging each
// store's own page and re-sorting by id DESC yields the true global page; fetching
// limit+1 per store guarantees the merged top-`limit` is complete.
// 한국어: 각 저장소에서 limit+1 후보를 가져와 전역 ID 내림차순으로 병합한다. 큰 source 하나가 다른 source의 최신 노드를 페이지에서 가리지 않게 한다.
func (s *ExplorationStore) ListByKindPageWithSources(kind string, before int64, limit int, q string) (nodes []*Node, hasMore bool, total int, err error) {
	if limit <= 0 {
		limit = 20
	}
	own, err := s.listByKindPageFiltered(kind, before, limit, q)
	if err != nil {
		return nil, false, 0, err
	}
	merged := own
	total, err = s.countByKindFiltered(kind, q)
	if err != nil {
		return nil, false, 0, err
	}

	sources, err := s.DirectSourceStores()
	if err != nil {
		return nil, false, 0, err
	}
	for _, source := range sources {
		page, err := source.Store.listByKindPageFiltered(kind, before, limit, q)
		if err != nil {
			return nil, false, 0, err
		}
		for _, n := range page {
			merged = append(merged, markInheritedNode(n, source.Task.TaskID))
		}
		cnt, err := source.Store.countByKindFiltered(kind, q)
		if err != nil {
			return nil, false, 0, err
		}
		total += cnt
	}

	sort.Slice(merged, func(i, j int) bool { return merged[i].ID > merged[j].ID })
	hasMore = len(merged) > limit
	if hasMore {
		merged = merged[:limit]
	}
	return merged, hasMore, total, nil
}

// GetNodeWithSources reads a node only when it belongs to this exploration or
// one of its direct sources. Inherited nodes are tagged so tool callers can keep
// them read-only and show their provenance.
// 한국어: 현재 또는 직접 source에 속한 노드만 찾고 source 출처를 표시한다. 상속 결과의 수정 금지는 호출 도구의 쓰기 경로와 함께 지켜야 한다.
func (s *ExplorationStore) GetNodeWithSources(id int64) (*Node, error) {
	node, err := s.GetNode(id)
	if err != nil || node != nil {
		return node, err
	}
	sources, err := s.DirectSourceStores()
	if err != nil {
		return nil, err
	}
	for _, source := range sources {
		node, err = source.Store.GetNode(id)
		if err != nil {
			return nil, err
		}
		if node != nil {
			if node.Kind == KindIntent && !inheritedIntentTerminal(node.State) {
				continue
			}
			return markInheritedNode(node, source.Task.TaskID), nil
		}
	}
	return nil, nil
}

// FindingIntentsWithSources combines finding lineage for the current
// exploration and each direct source. Node ids are globally unique.
func (s *ExplorationStore) FindingIntentsWithSources() (map[int64]int64, error) {
	out, err := s.FindingIntents()
	if err != nil {
		return nil, err
	}
	sources, err := s.DirectSourceStores()
	if err != nil {
		return nil, err
	}
	for _, source := range sources {
		items, err := source.Store.FindingIntentsTerminal()
		if err != nil {
			return nil, err
		}
		for findingID, intentID := range items {
			out[findingID] = intentID
		}
	}
	return out, nil
}

// ActivityTraceWithSources returns a work trace when its intent belongs to the
// current exploration or a direct source. It never searches indirect sources.
func (s *ExplorationStore) ActivityTraceWithSources(nodeID int64, limit int) ([]Activity, error) {
	acts, err := s.ActivityTrace(nodeID, limit)
	if err != nil || len(acts) > 0 {
		return acts, err
	}
	sources, err := s.DirectSourceStores()
	if err != nil {
		return nil, err
	}
	for _, source := range sources {
		node, nodeErr := source.Store.GetNode(nodeID)
		if nodeErr != nil {
			return nil, nodeErr
		}
		if node == nil || node.Kind != KindIntent {
			continue
		}
		acts, err = source.Store.ActivityTraceForTerminalIntent(nodeID, limit)
		if err != nil {
			return nil, err
		}
		if len(acts) > 0 {
			return markInheritedActivities(acts, source.Task.TaskID), nil
		}
	}
	return []Activity{}, nil
}

// ActivityListWithSources is the source-aware equivalent used by
// get_worker_output. Node ids are global, so the first owning exploration is
// unambiguous even when the work has not emitted any activity yet.
func (s *ExplorationStore) ActivityListWithSources(nodeID, sinceID int64, limit int) ([]Activity, int64, error) {
	node, err := s.GetNode(nodeID)
	if err != nil {
		return nil, sinceID, err
	}
	if node != nil {
		return s.ActivityList(&nodeID, sinceID, limit)
	}
	sources, err := s.DirectSourceStores()
	if err != nil {
		return nil, sinceID, err
	}
	for _, source := range sources {
		node, err = source.Store.GetNode(nodeID)
		if err != nil {
			return nil, sinceID, err
		}
		if node == nil || node.Kind != KindIntent {
			continue
		}
		acts, cursor, err := source.Store.ActivityListForTerminalIntent(nodeID, sinceID, limit)
		if err != nil {
			return nil, sinceID, err
		}
		return markInheritedActivities(acts, source.Task.TaskID), cursor, nil
	}
	return []Activity{}, sinceID, nil
}

// ActivityDetailWithSources keeps the legacy local-task lookup (including local
// thinking rows), while inherited details are restricted to terminal worker
// intents. Source planner/main rows have no node id and must never become part of
// inherited context.
// 한국어: 현재 작업의 상세 읽기는 기존 계약을 유지하고 source 상세는 종료된 worker intent로 제한한다. source planner/main 기록을 임의 activity ID로 끌어오지 않는다.
func (s *ExplorationStore) ActivityDetailWithSources(id int64) (string, error) {
	detail, err := s.ActivityDetail(id)
	if err != nil || detail != "" {
		return detail, err
	}
	acts, err := s.ActivityByIDsWithSources([]int64{id})
	if err != nil || len(acts) == 0 {
		return "", err
	}
	return acts[0].Detail, nil
}

// ActivityTraceSearchWithSources performs the scoped worker-trace search used
// by get_worker_trace. A node id resolves to at most one exploration because
// exploration node ids are global.
func (s *ExplorationStore) ActivityTraceSearchWithSources(nodeID int64, q string, limit int) ([]Activity, error) {
	acts, err := s.ActivityTraceSearch(&nodeID, q, limit)
	if err != nil || len(acts) > 0 {
		return acts, err
	}
	sources, err := s.DirectSourceStores()
	if err != nil {
		return nil, err
	}
	for _, source := range sources {
		node, nodeErr := source.Store.GetNode(nodeID)
		if nodeErr != nil {
			return nil, nodeErr
		}
		if node == nil || node.Kind != KindIntent {
			continue
		}
		acts, err = source.Store.ActivityTraceSearchForTerminalIntent(nodeID, q, limit)
		if err != nil {
			return nil, err
		}
		if len(acts) > 0 {
			return markInheritedActivities(acts, source.Task.TaskID), nil
		}
	}
	return []Activity{}, nil
}

// ActivityTraceSearchAllWithSources searches local worker traces plus every
// direct source. The local owner is excluded only from the current exploration;
// inherited traces are immutable historical context.
func (s *ExplorationStore) ActivityTraceSearchAllWithSources(excludeNodeID int64, q string, limit int) ([]Activity, error) {
	if limit <= 0 {
		limit = 100
	}
	out, err := s.ActivityTraceSearchExcluding(excludeNodeID, q, limit)
	if err != nil {
		return nil, err
	}
	sources, err := s.DirectSourceStores()
	if err != nil {
		return nil, err
	}
	for _, source := range sources {
		acts, err := source.Store.ActivityTraceSearchTerminalIntents(q, limit)
		if err != nil {
			return nil, err
		}
		out = append(out, markInheritedActivities(acts, source.Task.TaskID)...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// ActivityByIDsWithSources loads local step details with the legacy behavior. For
// direct sources it only returns rows attached to terminal intents, preventing an
// arbitrary global activity id from exposing source planner/main transcripts.
// 한국어: 각 ID의 소유 탐색과 종료 조건을 확인한다. 전역 ID 고유성을 이용해 연결하되 전역 조회 권한으로 해석하지 않는다.
func (s *ExplorationStore) ActivityByIDsWithSources(ids []int64) ([]Activity, error) {
	out, err := s.ActivityByIDs(ids)
	if err != nil {
		return nil, err
	}
	sources, err := s.DirectSourceStores()
	if err != nil {
		return nil, err
	}
	for _, source := range sources {
		acts, err := source.Store.ActivityByIDsForTerminalIntents(ids)
		if err != nil {
			return nil, err
		}
		out = append(out, markInheritedActivities(acts, source.Task.TaskID)...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// AssetRefsWithSources returns anchored nodes from this exploration and each
// direct source. Inherited entries retain their owning task id so API/UI callers
// can present them as immutable context.
// 한국어: 자산을 참조한 현재/직접 source의 노드를 출처와 함께 합친다. 데이터 복제 없이 원래 task ID를 유지하는 provenance 조회다.
func (s *ExplorationStore) AssetRefsWithSources(assetID int64) ([]AssetRef, error) {
	out, err := s.AssetRefs(assetID)
	if err != nil {
		return nil, err
	}
	sources, err := s.DirectSourceStores()
	if err != nil {
		return nil, err
	}
	for _, source := range sources {
		refs, err := source.Store.AssetRefs(assetID)
		if err != nil {
			return nil, err
		}
		for i := range refs {
			if refs[i].Kind == KindIntent && !inheritedIntentTerminal(refs[i].State) {
				continue
			}
			refs[i].SourceTaskID = source.Task.TaskID
			refs[i].Inherited = true
			out = append(out, refs[i])
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}
