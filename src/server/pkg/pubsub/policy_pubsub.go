// input: policy update events, agent subscription requests
// output: broadcast policy updates to subscribed agents via channels
// pos: pubsub - publish-subscribe mechanism for real-time policy distribution

package pubsub

import (
	"sync"

	policypb "github.com/haolipeng/ebpf-based-microsegment/api/proto/policy"
	"github.com/sirupsen/logrus"
)

// PolicyPubSub implements a publish-subscribe mechanism for policy updates
type PolicyPubSub struct {
	mu          sync.RWMutex
	subscribers map[string]chan *policypb.PolicyUpdate
}

// NewPolicyPubSub creates a new PolicyPubSub instance
func NewPolicyPubSub() *PolicyPubSub {
	return &PolicyPubSub{
		subscribers: make(map[string]chan *policypb.PolicyUpdate),
	}
}

// Subscribe adds a new subscriber for policy updates
// Returns a channel that will receive policy updates
func (p *PolicyPubSub) Subscribe(agentID string, bufferSize int) chan *policypb.PolicyUpdate {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Close and remove existing subscription if present
	if existingCh, exists := p.subscribers[agentID]; exists {
		close(existingCh)
		delete(p.subscribers, agentID)
		logrus.Warnf("Replaced existing subscription for agent: %s", agentID)
	}

	// Create new subscription channel
	ch := make(chan *policypb.PolicyUpdate, bufferSize)
	p.subscribers[agentID] = ch

	logrus.Infof("Agent %s subscribed to policy updates (buffer: %d)", agentID, bufferSize)
	return ch
}

// Unsubscribe removes a subscriber
func (p *PolicyPubSub) Unsubscribe(agentID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if ch, exists := p.subscribers[agentID]; exists {
		close(ch)
		delete(p.subscribers, agentID)
		logrus.Infof("Agent %s unsubscribed from policy updates", agentID)
	}
}

// Publish sends a policy update to all subscribers
// Non-blocking: if a subscriber's channel is full, the update is skipped for that subscriber
func (p *PolicyPubSub) Publish(update *policypb.PolicyUpdate) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	logrus.Debugf("Publishing policy update: type=%s, rule_id=%d, version=%d",
		update.UpdateType, update.Policy.GetRuleId(), update.PolicyVersion)

	for agentID, ch := range p.subscribers {
		select {
		case ch <- update:
			logrus.Debugf("Sent update to agent %s", agentID)
		default:
			logrus.Warnf("Agent %s channel is full, skipping update (version %d)",
				agentID, update.PolicyVersion)
		}
	}
}

// GetSubscriberCount returns the number of active subscribers
func (p *PolicyPubSub) GetSubscriberCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.subscribers)
}

// Close closes all subscriber channels and clears subscriptions
func (p *PolicyPubSub) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for agentID, ch := range p.subscribers {
		close(ch)
		logrus.Infof("Closed subscription for agent %s", agentID)
	}
	p.subscribers = make(map[string]chan *policypb.PolicyUpdate)
}
