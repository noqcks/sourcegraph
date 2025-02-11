import { test, expect } from '../../testing/integration'

test('shows suggestions', async ({ sg, page }) => {
    await page.goto('/search')
    const searchInput = page.getByRole('textbox')
    await searchInput.click()

    // Default suggestions
    await expect(page.getByLabel('Narrow your search')).toBeVisible()

    sg.mockTypes({
        SearchResults: () => ({
            repositories: [{ name: 'github.com/sourcegraph/sourcegraph' }],
            results: [
                {
                    __typename: 'FileMatch',
                    file: {
                        path: 'sourcegraph.md',
                        url: '',
                    },
                },
            ],
        }),
    })

    // Repo suggestions
    await searchInput.fill('source')
    await expect(page.getByLabel('Repositories')).toBeVisible()
    await expect(page.getByLabel('Files')).toBeVisible()

    // Fills suggestion
    await page.getByText('github.com/sourcegraph/sourcegraph').click()
    await expect(searchInput).toHaveText('repo:^github\\.com/sourcegraph/sourcegraph$ ')
})

test('submits search on enter', async ({ page }) => {
    await page.goto('/search')
    const searchInput = page.getByRole('textbox')
    await searchInput.fill('source')

    // Submit search
    await searchInput.press('Enter')
    await expect(page).toHaveURL(/\/search\?q=.+$/)
})

test('fills search query from URL', async ({ page }) => {
    await page.goto('/search?q=test')
    await expect(page.getByRole('textbox')).toHaveText('test')
})
