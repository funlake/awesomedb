package raftkv
import (
	"github.com/stretchr/testify/assert"
	"testing"
)
func TestNewStoppableListener(t *testing.T) {
	stopc := make(chan struct{})
	ln,err := NewStoppableListener("127.0.0.1:9989",stopc)
	assert.NoError(t,err)
	assert.NotNil(t,ln)
	_ = ln.Close()
}
