import React, { useRef } from 'react'

import classNames from 'classnames'
import MapSearchIcon from 'mdi-react/MapSearchIcon'
import { Route, RouteComponentProps, Switch } from 'react-router'

import { SiteSettingFields } from '@sourcegraph/shared/src/graphql-operations'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { PageHeader, LoadingSpinner } from '@sourcegraph/wildcard'

import { AuthenticatedUser } from '../auth'
import { withAuthenticatedUser } from '../auth/withAuthenticatedUser'
import { BatchChangesProps } from '../batches'
import { ErrorBoundary } from '../components/ErrorBoundary'
import { HeroPage } from '../components/HeroPage'
import { Page } from '../components/Page'

import { useFeatureFlag } from '../featureFlags/useFeatureFlag'
import { useUserExternalAccounts } from '../hooks/useUserExternalAccounts'
import { RouteDescriptor } from '../util/contributions'

import {
    maintenanceGroupHeaderLabel,
    maintenanceGroupInstrumentationItemLabel,
    maintenanceGroupMonitoringItemLabel,
    maintenanceGroupMigrationsItemLabel,
    maintenanceGroupUpdatesItemLabel,
    maintenanceGroupTracingItemLabel,
} from './sidebaritems'
import { SiteAdminSidebar, SiteAdminSideBarGroup, SiteAdminSideBarGroups } from './SiteAdminSidebar'

import styles from './SiteAdminArea.module.scss'

const NotFoundPage: React.ComponentType<React.PropsWithChildren<{}>> = () => (
    <HeroPage
        icon={MapSearchIcon}
        title="404: Not Found"
        subtitle="Sorry, the requested site admin page was not found."
    />
)

const NotSiteAdminPage: React.ComponentType<React.PropsWithChildren<{}>> = () => (
    <HeroPage icon={MapSearchIcon} title="403: Forbidden" subtitle="Only site admins are allowed here." />
)

export interface SiteAdminAreaRouteContext
    extends PlatformContextProps,
        SettingsCascadeProps,
        BatchChangesProps,
        TelemetryProps {
    site: Pick<SiteSettingFields, '__typename' | 'id'>
    authenticatedUser: AuthenticatedUser
    isLightTheme: boolean
    isSourcegraphDotCom: boolean

    /** This property is only used by {@link SiteAdminOverviewPage}. */
    overviewComponents: readonly React.ComponentType<React.PropsWithChildren<{}>>[]
}

export interface SiteAdminAreaRoute extends RouteDescriptor<SiteAdminAreaRouteContext> {}

interface SiteAdminAreaProps
    extends RouteComponentProps<{}>,
        PlatformContextProps,
        SettingsCascadeProps,
        BatchChangesProps,
        TelemetryProps {
    routes: readonly SiteAdminAreaRoute[]
    sideBarGroups: SiteAdminSideBarGroups
    overviewComponents: readonly React.ComponentType<React.PropsWithChildren<unknown>>[]
    authenticatedUser: AuthenticatedUser
    isLightTheme: boolean
    isSourcegraphDotCom: boolean
}

const sourcegraphOperatorSiteAdminMaintenanceBlockItems = [
    maintenanceGroupInstrumentationItemLabel,
    maintenanceGroupMonitoringItemLabel,
    maintenanceGroupMigrationsItemLabel,
    maintenanceGroupUpdatesItemLabel,
    maintenanceGroupTracingItemLabel,
]

const AuthenticatedSiteAdminArea: React.FunctionComponent<React.PropsWithChildren<SiteAdminAreaProps>> = props => {
    const reference = useRef<HTMLDivElement>(null)

    const { data: externalAccounts, loading: isExternalAccountsLoading } = useUserExternalAccounts(
        props.authenticatedUser.username
    )
    const [isSourcegraphOperatorSiteAdminHideMaintenance] = useFeatureFlag(
        'sourcegraph-operator-site-admin-hide-maintenance'
    )

    const adminSideBarGroups = React.useMemo(
        () =>
            props.sideBarGroups.reduce((groups, group) => {
                // DO NOT RETURN early in this function when reducing
                // curr is used to modify the current group in place, use cases:
                // - override the group with another one
                // - modify the items in the group
                // - assign null to curr to skip adding the group
                let curr = group

                // we default to hide o11y items if we are still trying to load external accounts
                // as long as the 'sourcegraph-operator-site-admin-hide-maintenance' is enabled
                // this is okay since such flag is only enabled on Cloud
                // for customer admin, those items are always invisble
                // for sourcegraph operator, they may notice some flickering during loading
                // this is okay as long as we do not impact customer admin experience
                if (isSourcegraphOperatorSiteAdminHideMaintenance) {
                    if (isExternalAccountsLoading) {
                        if (curr.header?.label === maintenanceGroupHeaderLabel) {
                            curr = {
                                ...curr,
                                items: curr.items.filter(
                                    item => !sourcegraphOperatorSiteAdminMaintenanceBlockItems.includes(item.label)
                                ),
                            }
                        }
                    } else {
                        if (!externalAccounts.some(account => account.serviceType === 'sourcegraph-operator')) {
                            if (curr.header?.label === maintenanceGroupHeaderLabel) {
                                curr = {
                                    ...curr,
                                    items: curr.items.filter(
                                        item => !sourcegraphOperatorSiteAdminMaintenanceBlockItems.includes(item.label)
                                    ),
                                }
                            }
                        }
                    }
                }

                if (curr === null) {
                    return groups
                }
                return [...groups, curr]
            }, [] as SiteAdminSideBarGroup[]),
        [
            props.sideBarGroups,
            isSourcegraphOperatorSiteAdminHideMaintenance,
            isExternalAccountsLoading,
            externalAccounts,
        ]
    )

    // If not site admin, redirect to sign in.
    if (!props.authenticatedUser.siteAdmin) {
        return <NotSiteAdminPage />
    }

    const context: SiteAdminAreaRouteContext = {
        authenticatedUser: props.authenticatedUser,
        platformContext: props.platformContext,
        settingsCascade: props.settingsCascade,
        isLightTheme: props.isLightTheme,
        isSourcegraphDotCom: props.isSourcegraphDotCom,
        batchChangesEnabled: props.batchChangesEnabled,
        batchChangesExecutionEnabled: props.batchChangesExecutionEnabled,
        batchChangesWebhookLogsEnabled: props.batchChangesWebhookLogsEnabled,
        site: { __typename: 'Site' as const, id: window.context.siteGQLID },
        overviewComponents: props.overviewComponents,
        telemetryService: props.telemetryService,
    }

    return (
        <Page>
            <PageHeader>
                <PageHeader.Heading as="h2" styleAs="h1">
                    <PageHeader.Breadcrumb>Admin</PageHeader.Breadcrumb>
                </PageHeader.Heading>
            </PageHeader>
            <div className="d-flex my-3 flex-column flex-sm-row" ref={reference}>
                <SiteAdminSidebar
                    className={classNames('flex-0 mr-3 mb-4', styles.sidebar)}
                    groups={adminSideBarGroups}
                    isSourcegraphDotCom={props.isSourcegraphDotCom}
                    batchChangesEnabled={props.batchChangesEnabled}
                    batchChangesExecutionEnabled={props.batchChangesExecutionEnabled}
                    batchChangesWebhookLogsEnabled={props.batchChangesWebhookLogsEnabled}
                />
                <div className="flex-bounded">
                    <ErrorBoundary location={props.location}>
                        <React.Suspense fallback={<LoadingSpinner className="m-2" />}>
                            <Switch>
                                {props.routes.map(
                                    ({ render, path, exact, condition = () => true }) =>
                                        condition(context) && (
                                            <Route
                                                // see https://github.com/ReactTraining/react-router/issues/4578#issuecomment-334489490
                                                key="hardcoded-key"
                                                path={props.match.url + path}
                                                exact={exact}
                                                render={routeComponentProps =>
                                                    render({ ...context, ...routeComponentProps })
                                                }
                                            />
                                        )
                                )}
                                <Route component={NotFoundPage} />
                            </Switch>
                        </React.Suspense>
                    </ErrorBoundary>
                </div>
            </div>
        </Page>
    )
}

/**
 * Renders a layout of a sidebar and a content area to display site admin information.
 */
export const SiteAdminArea = withAuthenticatedUser(AuthenticatedSiteAdminArea)
