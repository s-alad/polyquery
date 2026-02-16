package api

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/salad/polyquery/pq/pkg/store"
)

type Server struct {
	store *store.Queries
}

func New(s *store.Queries) *Server {
	return &Server{store: s}
}

func (s *Server) Register(app *fiber.App) {
	api := app.Group("/api")
	api.Get("/queue", s.getQueue)
	api.Post("/queue/:id/approve", s.approveQueue)
	api.Post("/queue/:id/reject", s.rejectQueue)
	api.Get("/markets", s.getMarkets)
	api.Get("/ledger", s.getLedger)
}

func (s *Server) getQueue(c fiber.Ctx) error {
	approvals, err := s.store.GetPendingApprovals(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(approvals)
}

func (s *Server) approveQueue(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	approval, err := s.store.GetApprovalByID(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	if err := s.store.ApproveApproval(c.Context(), id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	s.store.UpdateMarketStatus(c.Context(), store.UpdateMarketStatusParams{
		Status: "approved",
		ID:     approval.MarketID,
	})
	return c.JSON(fiber.Map{"status": "approved"})
}

func (s *Server) rejectQueue(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if err := s.store.RejectApproval(c.Context(), id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"status": "rejected"})
}

func (s *Server) getMarkets(c fiber.Ctx) error {
	status := c.Query("status", "watching")
	markets, err := s.store.GetMarketsByStatus(c.Context(), status)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(markets)
}

func (s *Server) getLedger(c fiber.Ctx) error {
	entries, err := s.store.GetLedger(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(entries)
}
