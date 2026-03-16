import type { Config } from "tailwindcss";

const config: Config = {
	content: ["./src/**/*.{html,js,svelte,ts}"],
	theme: {
		extend: {
			colors: {
				background: "rgb(var(--color-background) / <alpha-value>)",
				foreground: "rgb(var(--color-foreground) / <alpha-value>)",
				surface: "rgb(var(--color-surface) / <alpha-value>)",
				"surface-soft": "rgb(var(--color-surface-soft) / <alpha-value>)",
				border: "rgb(var(--color-border) / <alpha-value>)",
				muted: "rgb(var(--color-muted) / <alpha-value>)",
				accent: "rgb(var(--color-accent) / <alpha-value>)",
				"accent-foreground": "rgb(var(--color-accent-foreground) / <alpha-value>)",
				success: "rgb(var(--color-success) / <alpha-value>)"
			},
			boxShadow: {
				card: "0 16px 40px -24px rgba(0, 0, 0, 0.7)"
			},
			borderRadius: {
				xl: "0.9rem",
				"2xl": "1.2rem"
			},
			fontFamily: {
				sans: ["Sora", "ui-sans-serif", "system-ui", "Segoe UI", "sans-serif"]
			}
		}
	},
	plugins: []
};

export default config;
