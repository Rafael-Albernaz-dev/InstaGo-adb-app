package wailsapp

import (
	"context"

	domainlib "github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/library"
	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/usecase/library"
)

// LibraryBridge exposes R1 profile, managed library, integrity, and backup use cases to Wails 3 IPC.
type LibraryBridge struct {
	svc *library.Service
}

// NewLibraryBridge wraps library.Service for Wails 3 IPC binding.
func NewLibraryBridge(svc *library.Service) *LibraryBridge {
	return &LibraryBridge{svc: svc}
}

func (b *LibraryBridge) GetLibraryOverview(filter domainlib.LibraryFilter) (domainlib.LibraryOverview, error) {
	return b.svc.GetOverview(context.Background(), filter)
}

func (b *LibraryBridge) ImportBase(sourceDir string) (domainlib.ImportSummary, error) {
	return b.svc.ImportBase(context.Background(), sourceDir)
}

func (b *LibraryBridge) UpdateProfile(profile domainlib.CompanyProfile) (domainlib.CompanyProfile, error) {
	return b.svc.UpdateProfile(context.Background(), profile)
}

func (b *LibraryBridge) VerifyIntegrity() (domainlib.IntegrityReport, error) {
	return b.svc.VerifyIntegrity(context.Background())
}

func (b *LibraryBridge) CreateBackup() (domainlib.BackupResult, error) {
	return b.svc.CreateBackup(context.Background())
}

func (b *LibraryBridge) OpenFolder(scope string, assetID string) (string, error) {
	return b.svc.OpenFolder(context.Background(), scope, assetID)
}
