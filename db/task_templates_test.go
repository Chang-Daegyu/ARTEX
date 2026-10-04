package db

// 한국어 테스트 안내
// 템플릿 CRUD와 정규화 이름 중복 처리, 서로 다른 필드의 동시 PATCH 결합을 검증한다.
// 부분 갱신에서 전달하지 않은 필드는 유지해야 다른 편집자의 변경을 지우지 않는다.

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// 한국어 검증 목적: 템플릿 CRUD와 공백/대소문자를 정리한 이름 고유성 계약을 확인한다.
func TestTaskTemplateCRUDAndNormalizedUniqueness(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer d.Close()

	suffix := time.Now().UnixNano()
	name := fmt.Sprintf("Template %d", suffix)
	created, err := d.CreateTaskTemplate(TaskTemplateInput{
		Name:        "  " + strings.ReplaceAll(name, " ", "   ") + "  ",
		Description: "  initial description  ",
		Goal:        "  initial goal  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = d.DeleteTaskTemplate(created.ID) })
	if created.Name != name || created.Description != "initial description" || created.Goal != "initial goal" {
		t.Fatalf("template was not normalized: %+v", created)
	}

	if _, err := d.CreateTaskTemplate(TaskTemplateInput{
		Name: strings.ToUpper(name), Description: "duplicate", Goal: "duplicate",
	}); !errors.Is(err, ErrTaskTemplateNameConflict) {
		t.Fatalf("duplicate create error = %v, want %v", err, ErrTaskTemplateNameConflict)
	}

	got, err := d.GetTaskTemplate(created.ID)
	if err != nil || got == nil || got.Name != name {
		t.Fatalf("GetTaskTemplate = %+v, %v", got, err)
	}
	listed, err := d.ListTaskTemplates()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, template := range listed {
		if template.ID == created.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("created template %d missing from list", created.ID)
	}

	updatedName := name + " updated"
	updated, err := d.UpdateTaskTemplate(created.ID, TaskTemplateInput{
		Name: updatedName, Description: "new description", Goal: "new goal",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != updatedName || updated.Description != "new description" || updated.Goal != "new goal" {
		t.Fatalf("unexpected updated template: %+v", updated)
	}

	if _, err := d.CreateTaskTemplate(TaskTemplateInput{Name: "", Description: "x", Goal: "y"}); !errors.Is(err, ErrTaskTemplateInvalid) {
		t.Fatalf("empty name error = %v, want %v", err, ErrTaskTemplateInvalid)
	}
	deleted, err := d.DeleteTaskTemplate(created.ID)
	if err != nil || !deleted {
		t.Fatalf("DeleteTaskTemplate = %v, %v", deleted, err)
	}
	deleted, err = d.DeleteTaskTemplate(created.ID)
	if err != nil || deleted {
		t.Fatalf("second DeleteTaskTemplate = %v, %v", deleted, err)
	}
}

// 한국어 검증 목적: 서로 다른 필드를 바꾸는 두 부분 갱신이 서로의 변경을 지우지 않고 합쳐지는지 확인한다.
func TestTaskTemplateDisjointPatchesCompose(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v) - skipping", err)
	}
	defer d.Close()

	created, err := d.CreateTaskTemplate(TaskTemplateInput{
		Name:        fmt.Sprintf("Concurrent template %d", time.Now().UnixNano()),
		Description: "initial description",
		Goal:        "initial goal",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = d.DeleteTaskTemplate(created.ID) })

	description := "description from concurrent patch"
	goal := "goal from concurrent patch"
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, patch := range []TaskTemplatePatch{{Description: &description}, {Goal: &goal}} {
		patch := patch
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := d.PatchTaskTemplate(created.ID, patch)
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	got, err := d.GetTaskTemplate(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Description != description || got.Goal != goal {
		t.Fatalf("disjoint patches lost an update: %+v", got)
	}
}
