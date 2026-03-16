import { expect, test } from "@playwright/test";

test("catalog detail and saved flow", async ({ page }) => {
	await page.route("**/api/v1/resources**", async (route) => {
		await route.fulfill({
			status: 200,
			contentType: "application/json",
			body: JSON.stringify({
				items: [
					{
						id: "c01f9204-5f38-47e4-b3ec-c580691ff44f",
						slug: "dw-nicos-weg-a1-overview",
						title: "Nicos Weg A1 Overview",
						summary:
							"Structured beginner-friendly entry into practical German for daily communication.",
						sourceName: "Deutsche Welle",
						sourceType: "course",
						externalUrl: "https://learngerman.dw.com/en/nicos-weg/c-36519687",
						cefrLevel: "A1",
						format: "course",
						durationMinutes: 40,
						isFree: true,
						priceCents: null,
						skillTags: ["listening", "speaking"],
						topicTags: ["daily-life", "introductions"],
						isSaved: false
					}
				],
				count: 1
			})
		});
	});

	await page.addInitScript(() => {
		localStorage.clear();
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

	const savedIDs = await page.evaluate(() => {
		const raw = localStorage.getItem("deutsch.saved-resource-ids");
		return raw ? JSON.parse(raw) : [];
	});
	expect(savedIDs).toHaveLength(1);

	await page.getByRole("link", { name: /Saved/ }).click();

	await expect(page).toHaveURL(/\/saved/);
	await expect(page.getByRole("heading", { name: "Saved Resources" })).toBeVisible();
	await expect(page.getByRole("link", { name: selectedTitle })).toBeVisible();

	await page.getByRole("link", { name: selectedTitle }).first().click();
	await expect(page).toHaveURL(new RegExp(`${selectedHref}$`));
	await expect(page.getByRole("heading", { name: selectedTitle })).toBeVisible();
});
