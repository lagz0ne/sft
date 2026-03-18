import tailwindcss from "@tailwindcss/vite";
import { tanstackStart } from "@tanstack/react-start/plugin/vite";
import viteReact from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
	plugins: [tailwindcss(), tanstackStart({ spa: { enabled: true } }), viteReact()],
	resolve: {
		tsconfigPaths: true,
	},
	server: {
		port: 3001,
		proxy: {
			"/nats": {
				target: "ws://localhost:51741",
				ws: true,
			},
			"/a": {
				target: "http://localhost:51741",
			},
		},
	},
});
