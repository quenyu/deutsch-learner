import { expect, test } from "@playwright/test";

test("profile defaults and source-aware catalog filters", async ({ page }) => {
	await page.goto("/profile");
	await page.waitForLoadState("networkidle");

	await page.getByLabel("Display name").fill("Focused Learner");
	await page.getByLabel("Target level").selectOption("A1");

	const courseCheckbox = page.getByRole("checkbox", { name: "course", exact: true });
	if (!(await courseCheckbox.isChecked())) {
		await courseCheckbox.click();
	}
	const listeningCheckbox = page.getByRole("checkbox", { name: "listening", exact: true });
	if (!(await listeningCheckbox.isChecked())) {
		await listeningCheckbox.click();
	}
	const manualCheckbox = page.getByRole("checkbox", { name: "Manual Curation", exact: true });
	if (!(await manualCheckbox.isChecked())) {
		await manualCheckbox.click();
	}

	await page.getByRole("button", { name: "Save profile" }).click();
	await expect(page.getByText("Profile saved. Catalog defaults")).toBeVisible();

	await page.goto("/resources");
	await page.waitForLoadState("networkidle");
	await expect(page.getByText("Showing profile-based defaults")).toBeVisible();
	await expect(page.getByLabel("Provider")).toHaveValue("manual");

	const firstCard = page.getByTestId("resource-card").first();
	await expect(firstCard.getByText("manual", { exact: true }).first()).toBeVisible();
	await expect(firstCard.getByText("Curated")).toBeVisible();

	await page.getByLabel("CEFR level").selectOption("");
	await page.getByLabel("Skill").selectOption("");
	await page.getByLabel("Type").selectOption("article");
	await page.getByRole("button", { name: "Apply filters" }).click();
	await expect(page.getByRole("heading", { name: "German Case System Guide" })).toBeVisible();
});
