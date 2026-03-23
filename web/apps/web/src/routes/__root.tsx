import { Toaster } from "@sft-web/ui/components/sonner";
import { HeadContent, Outlet, Scripts, createRootRoute } from "@tanstack/react-router";

import { SpecProvider } from "../context/spec-context";
import { Lightbox } from "../components/lightbox";

import appCss from "../index.css?url";

export const Route = createRootRoute({
	head: () => ({
		meta: [
			{ charSet: "utf-8" },
			{ name: "viewport", content: "width=device-width, initial-scale=1" },
			{ title: "sft" },
		],
		links: [
			{ rel: "stylesheet", href: appCss },
			{ rel: "preconnect", href: "https://fonts.googleapis.com" },
			{
				rel: "preconnect",
				href: "https://fonts.gstatic.com",
				crossOrigin: "anonymous",
			},
			{
				rel: "stylesheet",
				href: "https://fonts.googleapis.com/css2?family=DM+Mono:wght@400;500&family=DM+Sans:ital,opsz,wght@0,9..40,400;0,9..40,500;0,9..40,600;0,9..40,700;1,9..40,400&family=Instrument+Serif:ital@0;1&display=swap",
			},
		],
	}),
	component: RootDocument,
});

function RootDocument() {
	return (
		<html lang="en">
			<head>
				<HeadContent />
			</head>
			<body className="h-svh overflow-hidden font-sans antialiased">
				<SpecProvider>
					<Outlet />
				</SpecProvider>
				<Lightbox />
				<Toaster richColors />
				<Scripts />
			</body>
		</html>
	);
}
