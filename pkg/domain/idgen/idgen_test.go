package idgen

import "testing"

func TestSnowflake_Generate(t *testing.T) {
	//if application run on kuberentes
	//this worker id can get from pod's name when run with statefulSet mode
	worker,err := NewSnowflake(0)
	if err != nil {
		t.Error(err)
	} else {
		t.Log(worker.Generate())
	}
}
