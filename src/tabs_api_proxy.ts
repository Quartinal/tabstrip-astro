// credits once again to https://github.com/r58Playz/tabstrip-dreamland for this file!

// @ts-ignore
import { loadTimeData } from './load_time_data.js'
// @ts-ignore
import {
  Tab,
  TabGroupVisualData,
  PageCallbackRouter,
} from './tab_strip.mojom-webui.js'

const STRINGS = {
  closeTab: 'Close tab',
  defaultTabTitle: 'Tab',
  tabGroupIdDataType: 'application/group-id',
  tabIdDataType: 'application/tab-id',
}

export enum CloseTabAction {
  CLOSE_BUTTON = 0,
  SWIPED_TO_CLOSE = 1,
}

export interface TabsApiProxy {
  activateTab(tabId: number): void

  getGroupVisualData(): Promise<{ data: { [id: string]: TabGroupVisualData } }>

  getTabs(): Promise<{ tabs: Tab[] }>

  closeTab(tabId: number, closeTabAction: CloseTabAction): void

  groupTab(tabId: number, groupId: string): void

  moveGroup(groupId: string, newIndex: number): void

  moveTab(tabId: number, newIndex: number): void

  setThumbnailTracked(tabId: number, thumbnailTracked: boolean): void

  ungroupTab(tabId: number): void

  isVisible(): boolean

  getLayout(): Promise<{ layout: { [key: string]: string } }>

  showEditDialogForGroup(
    groupId: string,
    locationX: number,
    locationY: number,
    width: number,
    height: number,
  ): void

  showTabContextMenu(tabId: number, locationX: number, locationY: number): void

  showBackgroundContextMenu(locationX: number, locationY: number): void

  closeContainer(): void

  reportTabActivationDuration(durationMs: number): void

  reportTabDataReceivedDuration(tabCount: number, durationMs: number): void

  reportTabCreationDuration(tabCount: number, durationMs: number): void

  getCallbackRouter(): PageCallbackRouter
}

let TABS_PROXY_SINGLETON: TabsApiProxyImpl | null = null

export class TabsApiProxyImpl extends EventTarget implements TabsApiProxy {
  tabId: number = 0
  tabs: Map<number, Tab> = new Map()
  layout: { [key: string]: string } = {}

  callbackRouter: PageCallbackRouter = new PageCallbackRouter()
  visibleHandler: () => boolean

  constructor(isVisible: () => boolean) {
    super()
    this.visibleHandler = isVisible
  }

  static createInstance(isVisible: () => boolean): TabsApiProxyImpl {
    if (TABS_PROXY_SINGLETON) {
      console.warn(
        'TabsApiProxyImpl already exists. Returning the existing instance.',
      )
      return TABS_PROXY_SINGLETON
    }
    console.log('Creating TabsApiProxyImpl instance...')
    loadTimeData.data = STRINGS
    TABS_PROXY_SINGLETON = new TabsApiProxyImpl(isVisible)
    return TABS_PROXY_SINGLETON
  }

  static getInstance(): TabsApiProxyImpl {
    if (!TABS_PROXY_SINGLETON) {
      console.error('No TabsApiProxyImpl created. Call createInstance() first.')
      throw new Error(
        'No TabsApiProxyImpl created. Ensure createInstance() is called before getInstance().',
      )
    }
    return TABS_PROXY_SINGLETON
  }

  dispatch(name: string, data: any) {
    this.dispatchEvent(new CustomEvent(name, { detail: data }))
  }

  setLayout(layout: { [key: string]: string }) {
    this.layout = layout
    this.callbackRouter.layoutChanged.notify(layout)
  }

  addTab(tab: Tab) {
    tab.id = this.tabId
    this.tabs.set(this.tabId, tab)
    this.tabId++
    this.callbackRouter.tabCreated.notify(tab)
  }

  activateTab(tabId: number): void {
    if (!this.tabs.has(tabId)) throw new Error('Invalid tab.')
    this.dispatch('activate', { tab: tabId })
  }

  async getGroupVisualData(): Promise<{
    data: { [id: string]: TabGroupVisualData }
  }> {
    console.warn('todo: getGroupVisualData')
    return { data: {} }
  }

  async getTabs(): Promise<{ tabs: Tab[] }> {
    return { tabs: Array.from(this.tabs.values()) }
  }

  closeTab(tabId: number, _closeTabAction: CloseTabAction): void {
    if (!this.tabs.has(tabId)) throw new Error('Invalid tab.')
    this.tabs.delete(tabId)
    this.dispatch('removeTab', { tab: tabId })
    this.callbackRouter.tabRemoved.notify(tabId)
  }

  groupTab(tabId: number, groupId: string): void {
    throw new Error('todo')
  }

  moveGroup(groupId: string, newIndex: number): void {
    throw new Error('todo')
  }

  moveTab(tabId: number, newIndex: number): void {
    if (!this.tabs.has(tabId)) throw new Error('Invalid tab.')
    this.tabs.get(tabId)!.index = newIndex
    this.dispatch('moveTab', { tab: tabId, index: newIndex })
  }

  ungroupTab(tabId: number): void {
    throw new Error('todo')
  }

  isVisible(): boolean {
    return this.visibleHandler()
  }

  async getLayout(): Promise<{ layout: { [key: string]: string } }> {
    return { layout: this.layout }
  }

  showEditDialogForGroup(
    groupId: string,
    locationX: number,
    locationY: number,
    width: number,
    height: number,
  ): void {
    throw new Error('todo')
  }

  showTabContextMenu(
    tabId: number,
    locationX: number,
    locationY: number,
  ): void {
    throw new Error('todo')
  }

  showBackgroundContextMenu(locationX: number, locationY: number): void {
    throw new Error('todo')
  }

  getCallbackRouter(): PageCallbackRouter {
    return this.callbackRouter
  }

  closeContainer(): void {
    // noop
  }

  setThumbnailTracked(tabId: number, thumbnailTracked: boolean): void {
    // noop
  }

  reportTabActivationDuration(durationMs: number): void {
    // noop
  }

  reportTabDataReceivedDuration(tabCount: number, durationMs: number): void {
    // noop
  }

  reportTabCreationDuration(tabCount: number, durationMs: number): void {
    // noop
  }
}
