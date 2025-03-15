package cluster

import (
	"testing"
	"time"

	"github.com/fyerfyer/fyer-cache/cache"
	"github.com/stretchr/testify/assert"
)

// 定义一个MsgAck常量用于测试
const MsgAck = 100

func TestNewTransport(t *testing.T) {
	tests := []struct {
		name   string
		config *TransportConfig
	}{
		{
			name:   "With nil config",
			config: nil,
		},
		{
			name: "With custom config",
			config: &TransportConfig{
				BindAddr:        ":8000",
				PathPrefix:      "/custom",
				Timeout:         10 * time.Second,
				MaxIdleConns:    20,
				MaxConnsPerHost: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewTransport(tt.config)
			assert.NotNil(t, tr)
			assert.NotNil(t, tr.config)
			assert.NotNil(t, tr.stopCh)

			if tt.config == nil {
				// 检查是否应用了默认配置
				assert.Equal(t, ":7946", tr.config.BindAddr)
				assert.Equal(t, "/cluster", tr.config.PathPrefix)
				assert.Equal(t, 5*time.Second, tr.config.Timeout)
				assert.Equal(t, 10, tr.config.MaxIdleConns)
				assert.Equal(t, 2, tr.config.MaxConnsPerHost)
			} else {
				// 检查是否应用了自定义配置
				assert.Equal(t, tt.config.BindAddr, tr.config.BindAddr)
				assert.Equal(t, tt.config.PathPrefix, tr.config.PathPrefix)
				assert.Equal(t, tt.config.Timeout, tr.config.Timeout)
				assert.Equal(t, tt.config.MaxIdleConns, tr.config.MaxIdleConns)
				assert.Equal(t, tt.config.MaxConnsPerHost, tr.config.MaxConnsPerHost)
			}
		})
	}
}

func TestTransport_SetHandler(t *testing.T) {
	tr := NewTransport(nil)
	assert.Nil(t, tr.handler, "Handler should be nil initially")

	// 测试设置处理器
	mockHandler := func(msg *cache.NodeMessage) (*cache.NodeMessage, error) {
		return &cache.NodeMessage{
			Type:     cache.MsgPong,
			SenderID: "test-node",
		}, nil
	}

	tr.SetHandler(mockHandler)
	assert.NotNil(t, tr.handler, "Handler should be set")

	// 测试更改处理器
	newMockHandler := func(msg *cache.NodeMessage) (*cache.NodeMessage, error) {
		return &cache.NodeMessage{
			Type:     MsgAck,
			SenderID: "new-test-node",
		}, nil
	}

	tr.SetHandler(newMockHandler)
	assert.NotNil(t, tr.handler, "Handler should be changed")
}

func TestTransport_HandleMessage(t *testing.T) {
	tests := []struct {
		name          string
		setupHandler  bool
		inputMessage  *cache.NodeMessage
		expectMessage *cache.NodeMessage
	}{
		{
			name:         "Without handler",
			setupHandler: false,
			inputMessage: &cache.NodeMessage{
				Type:     cache.MsgPing,
				SenderID: "node1",
				Payload:  []byte("ping"),
			},
			// 期望默认行为是回显相同的消息
			expectMessage: &cache.NodeMessage{
				Type:     cache.MsgPing,
				SenderID: "node1",
				Payload:  []byte("ping"),
			},
		},
		{
			name:         "With custom handler",
			setupHandler: true,
			inputMessage: &cache.NodeMessage{
				Type:     cache.MsgPing,
				SenderID: "node1",
				Payload:  []byte("ping"),
			},
			expectMessage: &cache.NodeMessage{
				Type:     cache.MsgPong,
				SenderID: "node2",
				Payload:  []byte("pong"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewTransport(nil)

			if tt.setupHandler {
				tr.SetHandler(func(msg *cache.NodeMessage) (*cache.NodeMessage, error) {
					// 返回自定义响应
					return tt.expectMessage, nil
				})
			}

			response, err := tr.HandleMessage(tt.inputMessage)

			assert.NoError(t, err)
			assert.NotNil(t, response)
			assert.Equal(t, tt.expectMessage.Type, response.Type)
			assert.Equal(t, tt.expectMessage.SenderID, response.SenderID)
			assert.Equal(t, tt.expectMessage.Payload, response.Payload)
		})
	}
}
