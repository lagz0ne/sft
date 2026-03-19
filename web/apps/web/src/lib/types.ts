export interface App {
	name: string;
	description: string;
}

export interface Transition {
	on_event: string;
	from_state?: string;
	to_state?: string;
	action?: string;
}

export interface Region {
	name: string;
	description?: string;
	events?: string[];
	transitions?: Transition[];
	attachments?: string[];
	regions?: Region[];
}

export interface Screen {
	name: string;
	description?: string;
	tags?: string[];
	attachments?: string[];
	regions?: Region[];
	transitions?: Transition[];
}

export interface Flow {
	name: string;
	description?: string;
	on_event?: string;
	sequence: string;
}

/** Parsed connection between two screens, derived from flow sequences */
export interface ScreenEdge {
	from: string;
	to: string;
	flow: string;
}

/** Position of a node on the canvas */
export interface NodePosition {
	x: number;
	y: number;
}

/** View state for overlay navigation */
export type ViewState =
	| { type: "canvas" }
	| { type: "screen"; screenName: string; highlightRegion?: string }
	| { type: "flow"; flowName: string };

/** Panel state within screen overlay */
export type PanelState =
	| { type: "none" }
	| { type: "state-machine"; regionName: string };

export interface Spec {
	app: App;
	screens: Screen[];
	flows?: Flow[];
}

export interface RenderElement {
	type: string;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	[key: string]: any;
}

export interface RenderSpec {
	elements?: Record<string, RenderElement>;
}
