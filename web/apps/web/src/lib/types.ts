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
	sequence: string;
}

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
