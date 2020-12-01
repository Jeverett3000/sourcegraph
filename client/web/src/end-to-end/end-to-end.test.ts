import { describe, test, before, beforeEach, after } from 'mocha'
import { afterEachSaveScreenshotIfFailed } from '../../../shared/src/testing/screenshotReporter'
import { afterEachRecordCoverage } from '../../../shared/src/testing/coverage'
import { createDriverForTest, Driver } from '../../../shared/src/testing/driver'
import MockDate from 'mockdate'
import { ExternalServiceKind } from '../../../shared/src/graphql/schema'
import { getConfig } from '../../../shared/src/testing/config'

const { gitHubToken, sourcegraphBaseUrl } = getConfig('gitHubToken', 'sourcegraphBaseUrl')

describe('e2e test suite', () => {
    let driver: Driver

    before(async function () {
        // Cloning the repositories takes ~1 minute, so give initialization 2
        // minutes instead of 1 (which would be inherited from
        // `jest.setTimeout(1 * 60 * 1000)` above).
        this.timeout(5 * 60 * 1000)

        // Reset date mocking
        MockDate.reset()

        const config = getConfig('headless', 'slowMo', 'testUserPassword')

        // Start browser
        driver = await createDriverForTest({
            sourcegraphBaseUrl,
            logBrowserConsole: true,
            ...config,
        })
        const clonedRepoSlugs = [
            'sourcegraph/java-langserver',
            'gorilla/mux',
            'gorilla/securecookie',
            'sourcegraph/jsonrpc2',
            'sourcegraph/go-diff',
            'sourcegraph/appdash',
            'sourcegraph/sourcegraph-typescript',
            'sourcegraph-testing/automation-e2e-test',
            'sourcegraph/e2e-test-private-repository',
        ]
        const alwaysCloningRepoSlugs = ['sourcegraphtest/AlwaysCloningTest']
        await driver.ensureLoggedIn({ username: 'test', password: config.testUserPassword, email: 'test@test.com' })
        await driver.resetUserSettings()
        await driver.ensureHasExternalService({
            kind: ExternalServiceKind.GITHUB,
            displayName: 'test-test-github',
            config: JSON.stringify({
                url: 'https://github.com',
                token: gitHubToken,
                repos: clonedRepoSlugs.concat(alwaysCloningRepoSlugs),
            }),
            ensureRepos: clonedRepoSlugs.map(slug => `github.com/${slug}`),
            alwaysCloning: alwaysCloningRepoSlugs.map(slug => `github.com/${slug}`),
        })
    })

    after('Close browser', () => driver?.close())

    afterEachSaveScreenshotIfFailed(() => driver.page)
    afterEachRecordCoverage(() => driver)

    beforeEach(async () => {
        if (driver) {
            // Clear local storage to reset sidebar selection (files or tabs) for each test
            await driver.page.evaluate(() => {
                localStorage.setItem('repo-revision-sidebar-last-tab', 'files')
            })

            await driver.resetUserSettings()
        }
    })

    describe('Core functionality', () => {
        test('Check settings are saved and applied', async () => {
            await driver.page.goto(sourcegraphBaseUrl + '/users/test/settings')
            await driver.page.waitForSelector('.test-settings-file .monaco-editor')

            const message = 'A wild notice appears!'
            await driver.replaceText({
                selector: '.test-settings-file .monaco-editor',
                newText: JSON.stringify({
                    notices: [
                        {
                            dismissable: false,
                            location: 'top',
                            message,
                        },
                    ],
                }),
                selectMethod: 'keyboard',
            })
            await driver.page.click('.test-settings-file .test-save-toolbar-save')
            await driver.page.waitForSelector('.test-global-alert .notices .global-alerts__alert', { visible: true })
            await driver.page.evaluate((message: string) => {
                const element = document.querySelector<HTMLElement>('.test-global-alert .notices .global-alerts__alert')
                if (!element) {
                    throw new Error('No .test-global-alert .notices .global-alerts__alert element found')
                }
                if (!element.textContent?.includes(message)) {
                    throw new Error(`Expected "${message}" message, but didn't find it`)
                }
            }, message)
        })

        test('Check allowed usernames', async () => {
            await driver.page.goto(sourcegraphBaseUrl + '/users/test/settings/profile')
            await driver.page.waitForSelector('.test-UserProfileFormFields-username')

            const name = 'alice.bob-chris-'

            await driver.replaceText({
                selector: '.test-UserProfileFormFields-username',
                newText: name,
                selectMethod: 'selectall',
            })

            await driver.page.click('#test-EditUserProfileForm__save')
            await driver.page.waitForSelector('.test-EditUserProfileForm__success', { visible: true })

            await driver.page.goto(sourcegraphBaseUrl + `/users/${name}/settings/profile`)
            await driver.replaceText({
                selector: '.test-UserProfileFormFields-username',
                newText: 'test',
                selectMethod: 'selectall',
            })

            await driver.page.click('#test-EditUserProfileForm__save')
            await driver.page.waitForSelector('.test-EditUserProfileForm__success', { visible: true })
        })
    })

    describe('External services', () => {
        test('External service add, edit, delete', async () => {
            const displayName = 'test-github-test-2'
            await driver.ensureHasExternalService({
                kind: ExternalServiceKind.GITHUB,
                displayName,
                config:
                    '{"url": "https://github.myenterprise.com", "token": "initial-token", "repositoryQuery": ["none"]}',
            })
            await driver.page.goto(sourcegraphBaseUrl + '/site-admin/external-services')
            await (
                await driver.page.waitForSelector(
                    `[data-test-external-service-name="${displayName}"] .test-edit-external-service-button`
                )
            ).click()

            // Type in a new external service configuration.
            await driver.replaceText({
                selector: '.test-external-service-editor .monaco-editor',
                newText:
                    '{"url": "https://github.myenterprise.com", "token": "second-token", "repositoryQuery": ["none"]}',
                selectMethod: 'selectall',
                enterTextMethod: 'paste',
            })
            await driver.page.click('.test-update-external-service-button')
            // Must wait for the operation to complete, or else a "Discard changes?" dialog will pop up
            await driver.page.waitForSelector('.test-update-external-service-button:not([disabled])', { visible: true })

            await (
                await driver.page.waitForSelector('.list-group-item[href="/site-admin/external-services"]', {
                    visible: true,
                })
            ).click()

            await Promise.all([
                driver.acceptNextDialog(),
                (
                    await driver.page.waitForSelector(
                        '[data-test-external-service-name="test-github-test-2"] .test-delete-external-service-button',
                        { visible: true }
                    )
                ).click(),
            ])

            await driver.page.waitFor(
                () => !document.querySelector('[data-test-external-service-name="test-github-test-2"]')
            )
        })

        test('External service repositoryPathPattern', async () => {
            const repo = 'sourcegraph/go-blame' // Tiny repo, fast to clone
            const repositoryPathPattern = 'foobar/{host}/{nameWithOwner}'
            const slug = `github.com/${repo}`
            const pathPatternSlug = `foobar/github.com/${repo}`

            const config = {
                kind: ExternalServiceKind.GITHUB,
                displayName: 'test-test-github-repoPathPattern',
                config: JSON.stringify({
                    url: 'https://github.com',
                    token: gitHubToken,
                    repos: [repo],
                    repositoryPathPattern,
                }),
                // Make sure repository is named according to path pattern
                ensureRepos: [pathPatternSlug],
            }
            await driver.ensureHasExternalService(config)

            // Make sure repository slug without path pattern redirects to path pattern
            await driver.page.goto(sourcegraphBaseUrl + '/' + slug)
            await driver.assertWindowLocationPrefix('/' + pathPatternSlug)
        })
    })
})
