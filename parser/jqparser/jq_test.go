package jqparser_test

import (
	"testing"

	"github.com/baruch/progress-wrap/parser/jqparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJQ_SimpleField(t *testing.T) {
	p, err := jqparser.New(".progress * 100")
	require.NoError(t, err)
	prog, found, err := p.Parse([]byte(`{"progress": 0.453}`))
	require.NoError(t, err)
	assert.True(t, found)
	assert.InDelta(t, 0.453, prog, 1e-9)
}

func TestJQ_Division(t *testing.T) {
	p, err := jqparser.New(".done / .total * 100")
	require.NoError(t, err)
	prog, found, err := p.Parse([]byte(`{"done": 45, "total": 100}`))
	require.NoError(t, err)
	assert.True(t, found)
	assert.InDelta(t, 0.45, prog, 1e-9)
}

func TestJQ_NestedField(t *testing.T) {
	p, err := jqparser.New(".status.progress * 100")
	require.NoError(t, err)
	prog, found, err := p.Parse([]byte(`{"status":{"progress":0.7}}`))
	require.NoError(t, err)
	assert.True(t, found)
	assert.InDelta(t, 0.70, prog, 1e-9)
}

func TestJQ_InvalidJSON(t *testing.T) {
	p, err := jqparser.New(".progress * 100")
	require.NoError(t, err)
	_, found, err := p.Parse([]byte("not json"))
	assert.Error(t, err)
	assert.False(t, found)
}

func TestJQ_FieldMissing(t *testing.T) {
	p, err := jqparser.New(".missing * 100")
	require.NoError(t, err)
	_, found, err := p.Parse([]byte(`{}`))
	// null * 100 = 0 in jq; we treat 0 as not-found
	require.NoError(t, err)
	assert.False(t, found)
}

func TestJQ_InvalidExpression(t *testing.T) {
	_, err := jqparser.New("[[[[invalid")
	assert.Error(t, err)
}
