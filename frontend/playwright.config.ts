import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
	testDir: "./tests",
	fullyParallel: true,
	retries: 0,
	use: {
		baseURL: "http://127.0.0.1:4173",
		trace: "on-first-retry"
	},
	webServer: [
		{
			command: "go run ./cmd/api",
			cwd: "../backend",
			env: {
				APP_PORT: "8081",
				DATA_BACKEND: "memory",
				REDIS_ADDR: ""
			},
			url: "http://127.0.0.1:8081/healthz",
			reuseExistingServer: true
		},
		{
			command: "npm run dev -- --host 127.0.0.1 --port 4173",
			env: {
				PUBLIC_API_BASE_URL: "",
				API_PROXY_TARGET: "http://127.0.0.1:8081"
			},
			url: "http://127.0.0.1:4173/resources",
			reuseExistingServer: true
		}
	],
	projects: [
		{
			name: "chromium",
			use: { ...devices["Desktop Chrome"] }
		}
	]
});
