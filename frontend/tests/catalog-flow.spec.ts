import { expect, test } from "@playwright/test";

const demoUserID = "11111111-1111-1111-1111-111111111111";
const resourceID = "c01f9204-5f38-47e4-b3ec-c580691ff44f";

test("catalog detail and saved flow", async ({ page }) => {
	await page.request.delete(`/api/v1/me/saved-resources/${resourceID}`, {
		headers: { "X-User-ID": demoUserID }
	});
	await page.request.put(`/api/v1/me/progress/${resourceID}`, {
		headers: {
			"Content-Type": "application/json",
			"X-User-ID": demoUserID
		},
		data: { status: "not_started" }
	});

	await page.goto("/resources");
	await page.waitForLoadState("networkidle");
	await page.getByRole("link", { name: /Saved/ }).click();
	await expect(page).toHaveURL(/\/saved/);
	await page.getByRole("navigation").getByRole("link", { name: "Catalog", exact: true }).click();
	await expect(page).toHaveURL(/\/resources/);
	await page.waitForLoadState("networkidle");

	const firstCard = page.getByTestId("resource-card").first();
	const selectedLink = firstCard.getByRole("link").first();
	const selectedTitle = (await selectedLink.textContent())?.trim() ?? "";

	await firstCard.getByRole("button", { name: "Save" }).click();
	await expect(firstCard.getByRole("button", { name: "Saved" })).toBeVisible();

	await page.getByRole("link", { name: /Saved/ }).click();
	await expect(page).toHaveURL(/\/saved/);
	await expect(page.getByRole("heading", { name: "Saved Resources" })).toBeVisible();
	await expect(page.getByRole("link", { name: selectedTitle })).toBeVisible();

	await page.getByRole("link", { name: selectedTitle }).first().click();
	await expect(page.getByRole("heading", { name: selectedTitle })).toBeVisible();

	await page.getByTestId("progress-in_progress").click();
	await expect(page.getByTestId("progress-in_progress")).toHaveAttribute("aria-pressed", "true");

	await page.getByRole("link", { name: /Saved/ }).click();
	await expect(page).toHaveURL(/\/saved/);
	await expect(page.locator('[data-testid="saved-in-progress"]').getByRole("link", { name: selectedTitle })).toBeVisible();
});
