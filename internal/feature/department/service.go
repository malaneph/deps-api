package department

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"deps-api/internal/model"
)

var (
	ErrNotFound      = errors.New("department not found")
	ErrMaxDepth      = errors.New("max department depth of 5 exceeded")
	ErrDuplicateName = errors.New("department name already exists among siblings")
	ErrSelfParent    = errors.New("department cannot be its own parent")
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List() ([]model.Department, error) {
	var departments []model.Department
	if err := s.db.Where("depth = ?", 1).Order("name ASC").Find(&departments).Error; err != nil {
		return nil, fmt.Errorf("list departments: %w", err)
	}
	return departments, nil
}

func (s *Service) GetByID(id uint, depth int) (*model.Department, error) {
	var dept model.Department
	if err := s.db.First(&dept, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get department %d: %w", id, err)
	}
	if depth > 0 {
		if err := s.loadChildren(&dept, depth); err != nil {
			return nil, err
		}
	}
	return &dept, nil
}

func (s *Service) loadChildren(dept *model.Department, depth int) error {
	if depth <= 0 {
		return nil
	}
	if err := s.db.Where("parent_id = ?", dept.ID).Order("name ASC").Find(&dept.Children).Error; err != nil {
		return fmt.Errorf("load children of %d: %w", dept.ID, err)
	}
	for i := range dept.Children {
		if err := s.loadChildren(&dept.Children[i], depth-1); err != nil {
			return err
		}
	}
	return nil
}

type CreateInput struct {
	Name     string
	ParentID *uint
}

func (s *Service) Create(input CreateInput) (*model.Department, error) {
	dept := model.Department{Name: input.Name}

	if input.ParentID != nil {
		parent, err := s.GetByID(*input.ParentID, 0)
		if err != nil {
			return nil, err
		}
		if parent.Depth >= 5 {
			return nil, ErrMaxDepth
		}
		dept.ParentID = input.ParentID
		dept.Depth = parent.Depth + 1
	}

	if err := s.db.Create(&dept).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrDuplicateName
		}
		return nil, fmt.Errorf("create department: %w", err)
	}
	return &dept, nil
}

type UpdateInput struct {
	Name       *string
	ParentID   *uint
	MoveToRoot bool
}

func (s *Service) Update(id uint, input UpdateInput) (*model.Department, error) {
	dept, err := s.GetByID(id, 0)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		dept.Name = *input.Name
	}

	reparenting := input.MoveToRoot || input.ParentID != nil
	if reparenting {
		if input.MoveToRoot {
			dept.ParentID = nil
			dept.Depth = 1
		} else {
			if *input.ParentID == id {
				return nil, ErrSelfParent
			}
			parent, err := s.GetByID(*input.ParentID, 0)
			if err != nil {
				return nil, err
			}
			if parent.Depth >= 5 {
				return nil, ErrMaxDepth
			}
			dept.ParentID = input.ParentID
			dept.Depth = parent.Depth + 1
		}
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(dept).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return ErrDuplicateName
			}
			return fmt.Errorf("update department: %w", err)
		}
		if reparenting {
			return updateDescendantDepths(tx, id, dept.Depth)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return dept, nil
}

func (s *Service) Delete(id uint) error {
	if _, err := s.GetByID(id, 0); err != nil {
		return err
	}
	if err := s.db.Delete(&model.Department{}, id).Error; err != nil {
		return fmt.Errorf("delete department: %w", err)
	}
	return nil
}

func updateDescendantDepths(tx *gorm.DB, parentID uint, parentDepth int) error {
	var children []model.Department
	if err := tx.Where("parent_id = ?", parentID).Find(&children).Error; err != nil {
		return fmt.Errorf("find children of %d: %w", parentID, err)
	}
	for _, child := range children {
		newDepth := parentDepth + 1
		if err := tx.Model(&child).Update("depth", newDepth).Error; err != nil {
			return fmt.Errorf("update depth for %d: %w", child.ID, err)
		}
		if err := updateDescendantDepths(tx, child.ID, newDepth); err != nil {
			return err
		}
	}
	return nil
}
