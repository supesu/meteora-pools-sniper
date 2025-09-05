package grpc

import (
	"context"
	"sync"

	pb "github.com/supesu/sniping-bot-v2/api/proto"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"github.com/supesu/sniping-bot-v2/pkg/logger"
)

// Streaming handles gRPC streaming functionality
type Streaming struct {
	logger           logger.Logger
	subscriptionRepo domain.SubscriptionRepository
	converter        *Converter

	eventStreams   map[string]chan *pb.TransactionEvent
	meteoraStreams map[string]chan *pb.MeteoraEvent
	mu             sync.RWMutex
}

// NewStreaming creates a new streaming instance
func NewStreaming(
	logger logger.Logger,
	subscriptionRepo domain.SubscriptionRepository,
	converter *Converter,
) *Streaming {
	return &Streaming{
		logger:           logger,
		subscriptionRepo: subscriptionRepo,
		converter:        converter,
		eventStreams:     make(map[string]chan *pb.TransactionEvent),
		meteoraStreams:   make(map[string]chan *pb.MeteoraEvent),
	}
}

// SubscribeToTransactions handles transaction streaming subscription
func (s *Streaming) SubscribeToTransactions(req *pb.SubscribeRequest, stream pb.SnipingService_SubscribeToTransactionsServer) error {
	s.logger.WithField("client_id", req.ClientId).Info("Client subscribed to transactions")

	// Create event channel for this client
	eventChan := make(chan *pb.TransactionEvent, EventChannelBufferSize)
	s.mu.Lock()
	s.eventStreams[req.ClientId] = eventChan
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.eventStreams, req.ClientId)
		s.mu.Unlock()
		close(eventChan)
	}()

	// Subscribe to programs
	for _, programID := range req.ProgramIds {
		err := s.subscriptionRepo.Subscribe(stream.Context(), req.ClientId, []string{programID})
		if err != nil {
			s.logger.WithError(err).WithField("program_id", programID).Error("Failed to subscribe to program")
			return err
		}
	}

	// Stream events to client
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case event := <-eventChan:
			if err := stream.Send(event); err != nil {
				s.logger.WithError(err).Error("Failed to send event to client")
				return err
			}
		}
	}
}

// SubscribeToMeteoraEvents handles Meteora event streaming subscription
func (s *Streaming) SubscribeToMeteoraEvents(req *pb.SubscribeRequest, stream pb.SnipingService_SubscribeToMeteoraEventsServer) error {
	s.logger.WithField("client_id", req.ClientId).Info("Client subscribed to Meteora events")

	// Create event channel for this client
	eventChan := make(chan *pb.MeteoraEvent, EventChannelBufferSize)
	s.mu.Lock()
	s.meteoraStreams[req.ClientId] = eventChan
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.meteoraStreams, req.ClientId)
		s.mu.Unlock()
		close(eventChan)
	}()

	// For now, subscribe to all Meteora events
	err := s.subscriptionRepo.Subscribe(stream.Context(), req.ClientId, []string{"meteora"})
	if err != nil {
		s.logger.WithError(err).Error("Failed to subscribe to Meteora events")
		return err
	}

	// Stream events to client
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case event := <-eventChan:
			if err := stream.Send(event); err != nil {
				s.logger.WithError(err).Error("Failed to send Meteora event to client")
				return err
			}
		}
	}
}

// notifySubscribers sends transaction events to all subscribed clients
func (s *Streaming) notifySubscribers(ctx context.Context, tx *pb.Transaction) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	event := &pb.TransactionEvent{
		Transaction: tx,
		EventType:   "transaction_processed",
		EventTime:   tx.Timestamp,
	}

	for clientID, eventChan := range s.eventStreams {
		select {
		case eventChan <- event:
			s.logger.WithField("client_id", clientID).Debug("Sent transaction event to client")
		default:
			s.logger.WithField("client_id", clientID).Warn("Client event channel is full, dropping event")
		}
	}
}

// notifyMeteoraSubscribers sends Meteora events to all subscribed clients
func (s *Streaming) notifyMeteoraSubscribers(ctx context.Context, event *pb.MeteoraEvent) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for clientID, eventChan := range s.meteoraStreams {
		select {
		case eventChan <- event:
			s.logger.WithField("client_id", clientID).Debug("Sent Meteora event to client")
		default:
			s.logger.WithField("client_id", clientID).Warn("Client Meteora event channel is full, dropping event")
		}
	}
}
