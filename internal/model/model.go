package model

type App struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Screen struct {
	ID          int64  `json:"id"`
	AppID       int64  `json:"app_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Entry       bool   `json:"entry"`
}

type Region struct {
	ID                int64  `json:"id"`
	AppID             int64  `json:"app_id"`
	ParentType        string `json:"parent_type"`
	ParentID          int64  `json:"parent_id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	DiscoveryLayout   string `json:"discovery_layout,omitempty"`
	DeliveryClasses   string `json:"delivery_classes,omitempty"`
	DeliveryComponent string `json:"delivery_component,omitempty"`
}

type Tag struct {
	ID         int64  `json:"id"`
	EntityType string `json:"entity_type"`
	EntityID   int64  `json:"entity_id"`
	Tag        string `json:"tag"`
}

type Event struct {
	ID         int64  `json:"id"`
	RegionID   int64  `json:"region_id"`
	Name       string `json:"name"`
	Annotation string `json:"annotation,omitempty"`
}

type Transition struct {
	ID        int64  `json:"id"`
	OwnerType string `json:"owner_type"`
	OwnerID   int64  `json:"owner_id"`
	OnEvent   string `json:"on_event"`
	FromState string `json:"from_state,omitempty"`
	ToState   string `json:"to_state,omitempty"`
	Action    string `json:"action,omitempty"`
}

// Phase 4: State machine templates

type StateTemplate struct {
	ID         int64  `json:"id"`
	AppID      int64  `json:"app_id"`
	Name       string `json:"name"`
	Definition string `json:"definition"`
}

// Phase 3: Fixtures

type Fixture struct {
	ID      int64  `json:"id"`
	AppID   int64  `json:"app_id"`
	Name    string `json:"name"`
	Extends string `json:"extends,omitempty"`
	Data    string `json:"data"`
}

type StateFixture struct {
	ID          int64  `json:"id"`
	OwnerType   string `json:"owner_type"`
	OwnerID     int64  `json:"owner_id"`
	StateName   string `json:"state_name"`
	FixtureName string `json:"fixture_name"`
}

// Phase 5: State-region visibility

type StateRegion struct {
	ID         int64  `json:"id"`
	OwnerType  string `json:"owner_type"`
	OwnerID    int64  `json:"owner_id"`
	StateName  string `json:"state_name"`
	RegionName string `json:"region_name"`
}

// Discovery/Delivery layout model

type Layout struct {
	ID      int64  `json:"id"`
	AppID   int64  `json:"app_id"`
	Name    string `json:"name"`
	Classes string `json:"classes"` // JSON array of Tailwind classes
}

// Phase 5: Enums

type Enum struct {
	ID     int64  `json:"id"`
	AppID  int64  `json:"app_id"`
	Name   string `json:"name"`
	Values string `json:"values"`
}

// Phase 2: Data model types

type DataType struct {
	ID     int64  `json:"id"`
	AppID  int64  `json:"app_id"`
	Name   string `json:"name"`
	Fields string `json:"fields"`
}

type ContextField struct {
	ID        int64  `json:"id"`
	OwnerType string `json:"owner_type"`
	OwnerID   int64  `json:"owner_id"`
	FieldName string `json:"field_name"`
	FieldType string `json:"field_type"`
}

type AmbientRef struct {
	ID        int64  `json:"id"`
	RegionID  int64  `json:"region_id"`
	LocalName string `json:"local_name"`
	Source    string `json:"source"`
	Query    string `json:"query"`
}

type RegionData struct {
	ID        int64  `json:"id"`
	RegionID  int64  `json:"region_id"`
	FieldName string `json:"field_name"`
	FieldType string `json:"field_type"`
}

// v2: Entities, Experiments, Component Schemas

type Entity struct {
	ID    int64  `json:"id"`
	AppID int64  `json:"app_id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Data  string `json:"data"`
}

type Experiment struct {
	ID          int64  `json:"id"`
	AppID       int64  `json:"app_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Scope       string `json:"scope"`
	Overlay     string `json:"overlay"`
	Status      string `json:"status"`
}

type ComponentSchema struct {
	ID    int64  `json:"id"`
	AppID int64  `json:"app_id"`
	Name  string `json:"name"`
	Props string `json:"props"`
}
