package timeline

// StoreType identifies the storage backend
type StoreType string

const (
	StoreTypeMemory StoreType = "memory"
	StoreTypeSQLite StoreType = "sqlite"
)

// StoreConfig holds configuration for the event store
type StoreConfig struct {
	Type    StoreType
	Path    string // For SQLite: database file path
	MaxSize int    // For Memory: ring buffer size
}

// DefaultStoreConfig returns sensible defaults
func DefaultStoreConfig() StoreConfig {
	return StoreConfig{
		Type:    StoreTypeMemory,
		MaxSize: 1000,
	}
}
