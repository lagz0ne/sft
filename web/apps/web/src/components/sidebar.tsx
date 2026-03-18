import { Link, useLocation } from "@tanstack/react-router";
import { useSpecContext } from "../context/spec-context";

export function Sidebar() {
	const { spec, loading } = useSpecContext();
	const location = useLocation();

	if (loading || !spec) {
		return (
			<aside className="flex w-[260px] flex-col border-r border-border bg-background">
				<div className="border-b border-border p-4">
					<div className="h-5 w-20 animate-pulse rounded bg-muted" />
				</div>
			</aside>
		);
	}

	return (
		<aside className="flex w-[260px] shrink-0 flex-col overflow-hidden border-r border-border bg-background">
			<div className="shrink-0 border-b border-border px-4 py-5">
				<h1 className="text-base font-bold leading-tight tracking-tight">{spec.app.name}</h1>
				{spec.app.description && (
					<p className="mt-1 line-clamp-2 text-xs leading-normal text-muted-foreground">
						{spec.app.description}
					</p>
				)}
			</div>

			<nav className="flex-1 overflow-y-auto py-2">
				<div className="px-4 pb-1 pt-2.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
					Screens
				</div>
				<NavItem to="/" label="Overview" active={location.pathname === "/"} />
				{spec.screens.map((screen) => (
					<NavItem
						key={screen.name}
						to="/screens/$name"
						params={{ name: screen.name }}
						label={screen.name}
						active={location.pathname === `/screens/${encodeURIComponent(screen.name)}`}
						dot={!!screen.attachments?.length}
					/>
				))}

				{spec.flows && spec.flows.length > 0 && (
					<>
						<div className="mt-2 px-4 pb-1 pt-2.5 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
							Flows
						</div>
						{spec.flows.map((flow) => (
							<NavItem
								key={flow.name}
								to="/flows/$name"
								params={{ name: flow.name }}
								label={flow.name}
								active={location.pathname === `/flows/${encodeURIComponent(flow.name)}`}
							/>
						))}
					</>
				)}
			</nav>
		</aside>
	);
}

function NavItem({
	to,
	params,
	label,
	active,
	dot,
}: {
	to: string;
	params?: Record<string, string>;
	label: string;
	active: boolean;
	dot?: boolean;
}) {
	return (
		<Link
			to={to}
			params={params}
			className={`flex w-full items-center gap-2 px-4 py-1.5 text-left text-sm leading-snug transition-colors ${
				active
					? "bg-accent font-semibold text-accent-foreground"
					: "text-muted-foreground hover:bg-muted hover:text-foreground"
			}`}
		>
			<span className="truncate">{label}</span>
			{dot && <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-primary opacity-50" />}
		</Link>
	);
}
