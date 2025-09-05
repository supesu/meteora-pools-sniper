package grpc

import (
	pb "github.com/supesu/sniping-bot-v2/api/proto"
	"github.com/supesu/sniping-bot-v2/pkg/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Converter handles protobuf/domain conversions
type Converter struct{}

// NewConverter creates a new converter
func NewConverter() *Converter {
	return &Converter{}
}

// convertProtobufToDomain converts protobuf transaction to domain
func (c *Converter) convertProtobufToDomain(tx *pb.Transaction) *domain.Transaction {
	return &domain.Transaction{
		Signature: tx.Signature,
		ProgramID: tx.ProgramId,
		Accounts:  tx.Accounts,
		Data:      tx.Data,
		Timestamp: tx.Timestamp.AsTime(),
		Slot:      tx.Slot,
		Status:    c.convertProtobufStatus(tx.Status),
	}
}

// convertDomainToProtobuf converts domain transaction to protobuf
func (c *Converter) convertDomainToProtobuf(tx *domain.Transaction) *pb.Transaction {
	return &pb.Transaction{
		Signature: tx.Signature,
		ProgramId: tx.ProgramID,
		Accounts:  tx.Accounts,
		Data:      tx.Data,
		Timestamp: timestamppb.New(tx.Timestamp),
		Slot:      tx.Slot,
		Status:    c.convertDomainStatus(tx.Status),
	}
}

// convertProtobufStatus converts protobuf status to domain
func (c *Converter) convertProtobufStatus(pbStatus pb.TransactionStatus) domain.TransactionStatus {
	switch pbStatus {
	case pb.TransactionStatus_TRANSACTION_STATUS_CONFIRMED:
		return domain.TransactionStatusConfirmed
	case pb.TransactionStatus_TRANSACTION_STATUS_FINALIZED:
		return domain.TransactionStatusFinalized
	case pb.TransactionStatus_TRANSACTION_STATUS_FAILED:
		return domain.TransactionStatusFailed
	default:
		return domain.TransactionStatusUnknown
	}
}

// convertDomainStatus converts domain status to protobuf
func (c *Converter) convertDomainStatus(domainStatus domain.TransactionStatus) pb.TransactionStatus {
	switch domainStatus {
	case domain.TransactionStatusConfirmed:
		return pb.TransactionStatus_TRANSACTION_STATUS_CONFIRMED
	case domain.TransactionStatusFinalized:
		return pb.TransactionStatus_TRANSACTION_STATUS_FINALIZED
	case domain.TransactionStatusFailed:
		return pb.TransactionStatus_TRANSACTION_STATUS_FAILED
	default:
		return pb.TransactionStatus_TRANSACTION_STATUS_UNKNOWN
	}
}

// convertProtobufEventToDomain converts protobuf Meteora event to domain
func (c *Converter) convertProtobufEventToDomain(event *pb.MeteoraEvent) *domain.MeteoraPoolEvent {
	return &domain.MeteoraPoolEvent{
		PoolAddress:   event.PoolInfo.PoolAddress,
		TokenAMint:    event.PoolInfo.TokenAMint,
		TokenBMint:    event.PoolInfo.TokenBMint,
		TokenASymbol:  event.PoolInfo.TokenASymbol,
		TokenBSymbol:  event.PoolInfo.TokenBSymbol,
		CreatorWallet: event.PoolInfo.CreatorWallet,
		CreatedAt:     event.EventTime.AsTime(),
		Slot:          event.PoolInfo.Slot,
	}
}
