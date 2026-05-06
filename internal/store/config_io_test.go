package store

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestExportImportRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(context.Background(), Options{DBPath: filepath.Join(dir, "a.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	dsName := "ds-1"
	dsType := "local"
	view, err := s.CreateDataSource(context.Background(), DataSourceInput{Name: &dsName, Type: &dsType})
	if err != nil {
		t.Fatalf("CreateDataSource: %v", err)
	}

	cfgRaw := json.RawMessage([]byte(`{"title":"CPU","unit":"%"}`))
	mqRaw := json.RawMessage([]byte(`{"path":"system/cpu/total","shape":"Scalar"}`))
	wType := "bignumber"
	x, y, w, h := 0, 0, 16, 10
	dsID := view.ID
	icon := "ICONIFY"
	iconV := "mdi:numeric"
	if _, err := s.CreateWidget(context.Background(), WidgetInput{
		Type: &wType, X: &x, Y: &y, W: &w, H: &h,
		IconType: &icon, IconValue: &iconV,
		DataSourceID: &dsID,
		MetricQuery:  &mqRaw,
		Config:       &cfgRaw,
	}); err != nil {
		t.Fatalf("CreateWidget: %v", err)
	}

	bundle, err := s.ExportConfig(context.Background())
	if err != nil {
		t.Fatalf("ExportConfig: %v", err)
	}
	if bundle.Version != ConfigBundleVersion {
		t.Errorf("version = %d", bundle.Version)
	}
	if len(bundle.DataSources) != 1 || len(bundle.Widgets) != 1 {
		t.Fatalf("bundle = %+v", bundle)
	}

	// Rename datasource then re-import resets state.
	dir2 := t.TempDir()
	s2, _ := Open(context.Background(), Options{DBPath: filepath.Join(dir2, "b.db")})
	defer s2.Close()

	summary, err := s2.ImportConfig(context.Background(), bundle)
	if err != nil {
		t.Fatalf("ImportConfig: %v", err)
	}
	if !summary.BoardUpdated || summary.DataSourcesAdded != 1 || summary.WidgetsAdded != 1 {
		t.Errorf("summary = %+v", summary)
	}

	dsList, _ := s2.ListDataSources(context.Background())
	if len(dsList) != 1 {
		t.Fatalf("expected 1 ds after import, got %d", len(dsList))
	}
	wList, _ := s2.ListWidgets(context.Background(), DefaultBoardID)
	if len(wList) != 1 {
		t.Fatalf("expected 1 widget after import, got %d", len(wList))
	}
	if wList[0].DataSourceID == nil || *wList[0].DataSourceID != dsList[0].ID {
		t.Errorf("widget data_source_id remap failed: got %v want %d", wList[0].DataSourceID, dsList[0].ID)
	}
}

func TestImportInvalidVersion(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(context.Background(), Options{DBPath: filepath.Join(dir, "c.db")})
	defer s.Close()

	_, err := s.ImportConfig(context.Background(), &ConfigBundle{Version: 99})
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
}
