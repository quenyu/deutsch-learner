import { expect, test } from "@playwright/test";

test("catalog detail and saved flow", async ({ page }) => {
	const resource = {
		id: "c01f9204-5f38-47e4-b3ec-c580691ff44f",
		slug: "dw-nicos-weg-a1-overview",
		title: "Nicos Weg A1 Overview",
		summary: "Structured beginner-friendly entry into practical German for daily communication.",
		sourceName: "Deutsche Welle",
		sourceType: "course",
		externalUrl: "https://learngerman.dw.com/en/nicos-weg/c-36519687",
		cefrLevel: "A1",
		format: "course",
		durationMinutes: 40,
		isFree: true,
		priceCents: null,
		skillTags: ["listening", "speaking"],
		topicTags: ["daily-life", "introductions"]
	};

	let saved = false;

	await page.route("**/api/v1/**", async (route) => {
		const request = route.request();
		const url = new URL(request.url());

		if (request.method() === "GET" && url.pathname === "/api/v1/resources") {
			await route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({
					items: [{ ...resource, isSaved: saved }],
					count: 1
				})
			});
			return;
		}

		if (request.method() === "GET" && url.pathname === `/api/v1/resources/${resource.slug}`) {
			await route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({ ...resource, isSaved: saved })
			});
			return;
		}

		if (request.method() === "GET" && url.pathname === "/api/v1/me/saved-resources") {
			await route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({
					items: saved ? [{ ...resource, isSaved: true }] : [],
					count: saved ? 1 : 0
				})
			});
			return;
		}

		if (request.method() === "POST" && url.pathname === "/api/v1/me/saved-resources") {
			saved = true;
			await route.fulfill({
				status: 201,
				contentType: "application/json",
				body: JSON.stringify({
					saved: true,
					created: true
				})
			});
			return;
		}

		if (request.method() === "DELETE" && url.pathname === `/api/v1/me/saved-resources/${resource.id}`) {
			saved = false;
			await route.fulfill({
				status: 200,
				contentType: "application/json",
				body: JSON.stringify({
					saved: false,
					removed: true
				})
			});
			return;
		}

		await route.fulfill({
			status: 404,
			contentType: "application/json",
			body: JSON.stringify({ message: "not found" })
		});
	});

	await page.goto("/resources");
	await expect(page.getByRole("heading", { name: "Resource Catalog" })).toBeVisible();

	const firstCard = page.getByTestId("resource-card").first();
	const selectedLink = firstCard.getByRole("link").first();
	const selectedTitle = (await selectedLink.textContent())?.trim() ?? "";
	const selectedHref = (await selectedLink.getAttribute("href")) ?? "";
	const saveButton = firstCard.getByRole("button", { name: "Save" });

	await saveButton.click();
	await expect(firstCard.getByRole("button", { name: "Saved" })).toBeVisible();

	await page.getByRole("link", { name: /Saved/ }).click();

	await expect(page).toHaveURL(/\/saved/);
	await expect(page.getByRole("heading", { name: "Saved Resources" })).toBeVisible();
	await expect(page.getByRole("link", { name: selectedTitle })).toBeVisible();

	await page.reload();
	await expect(page.getByRole("link", { name: selectedTitle })).toBeVisible();

	await page.getByRole("link", { name: selectedTitle }).first().click();
	await expect(page).toHaveURL(new RegExp(`${selectedHref}$`));
	await expect(page.getByRole("heading", { name: selectedTitle })).toBeVisible();
});
