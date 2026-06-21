<template>
  <div class="h-full flex flex-col px-5 pt-5 overflow-y-auto [&::-webkit-scrollbar]:hidden">
    <div class="pb-3 z-40 space-y-2" id="header" style="--wails-draggable:no-drag">
      <div class="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-2">
        <div
            class="inline-flex h-10 shrink-0 items-center gap-2 rounded-md border px-3 shadow-sm transition-colors"
            :class="isProxy ? 'border-green-200 bg-green-50' : 'border-gray-200 bg-white'"
        >
          <NSwitch size="medium" :value="isProxy" @update:value="toggleProxy" />
          <span class="inline-flex items-center gap-1.5 whitespace-nowrap text-sm font-medium" :class="isProxy ? 'text-green-700' : 'text-gray-800'">
            <span v-if="isProxy" class="inline-block h-1.5 w-1.5 rounded-full bg-green-600 animate-pulse"></span>
            {{ isProxy ? t("index.grabbing") : t("index.open_grab") }}
          </span>
          <span v-if="data.length > 0" class="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-semibold text-gray-600">{{ data.length }}</span>
        </div>

        <div class="inline-flex min-h-10 shrink-0 flex-wrap items-center gap-x-3 gap-y-2 rounded-md border border-gray-200 bg-gray-50/70 px-3 py-1">
          <span class="mr-1 whitespace-nowrap text-sm text-gray-500">{{ t('index.grab_type') }}</span>
          <NCheckbox
              v-for="item in visibleClassify"
              :key="item.value"
              :checked="isResourceTypeActive(item.value)"
              @update:checked="(checked: boolean) => updateResourceType(item.value, checked)"
          >
            {{ getClassifyLabel(item) }}
          </NCheckbox>
          <NPopover v-if="moreClassify.length > 0" placement="bottom-start" trigger="click">
            <template #trigger>
              <NButton size="small" quaternary>
                {{ t('index.more_types') }}{{ selectedMoreClassifyCount > 0 ? `(${selectedMoreClassifyCount})` : '' }}
              </NButton>
            </template>
            <div class="grid min-w-[180px] grid-cols-2 gap-x-4 gap-y-2 p-1">
              <NCheckbox
                  v-for="item in moreClassify"
                  :key="item.value"
                  :checked="isResourceTypeActive(item.value)"
                  @update:checked="(checked: boolean) => updateResourceType(item.value, checked)"
              >
                {{ getClassifyLabel(item) }}
              </NCheckbox>
            </div>
          </NPopover>
        </div>

        <div class="ml-auto flex min-h-10 min-w-[420px] flex-[1_1_480px] items-center gap-2 rounded-md border border-gray-200 bg-gray-50/70 px-3 py-1">
          <span class="whitespace-nowrap text-sm text-gray-500">{{ t('setting.save_dir') }}</span>
          <div
              class="flex min-h-8 min-w-0 flex-1 items-start rounded border border-gray-200 bg-white px-2 py-1 text-xs leading-4 text-gray-800"
              :title="store.globalConfig.SaveDirectory"
          >
            <span
                v-if="store.globalConfig.SaveDirectory"
                class="block min-w-0 flex-1 whitespace-normal break-all text-left"
            >{{ store.globalConfig.SaveDirectory }}</span>
            <span v-else class="text-gray-400">{{ t('index.save_path_empty') }}</span>
          </div>
          <NButton size="small" @click.stop="selectDownloadDir">
            <template #icon>
              <n-icon><FolderOpenOutline/></n-icon>
            </template>
            {{ t('common.select') }}
          </NButton>
          <NButton size="small" @click.stop="openSaveDirectory">
            <template #icon>
              <n-icon><OpenOutline/></n-icon>
            </template>
            {{ t('index.open_save_dir') }}
          </NButton>
        </div>
      </div>

      <div class="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-2">
        <NButton quaternary type="primary" @click.stop="batchExtractCaption">
          <template #icon>
            <n-icon><DocumentTextOutline/></n-icon>
          </template>
          {{ t('index.batch_extract_caption') }}
        </NButton>
        <NButton quaternary type="primary" @click.stop="batchDown">
          <template #icon>
            <n-icon><DownloadOutline/></n-icon>
          </template>
          {{ t('index.batch_download') }}
        </NButton>
        <NButton quaternary type="error" @click.stop="batchCancel">
          <template #icon>
            <n-icon><CloseOutline/></n-icon>
          </template>
          {{ t('index.cancel_down') }}
        </NButton>

        <NButton v-if="rememberChoice" quaternary type="error" @click.stop="clear">
          <template #icon>
            <n-icon><TrashOutline/></n-icon>
          </template>
          {{ t("index.clear_list") }}
        </NButton>
        <n-popconfirm
            v-else
            @positive-click="()=>{rememberChoice=rememberChoiceTmp;clear()}"
            :show-icon="false"
        >
          <template #trigger>
            <NButton quaternary type="error" @click="selectClearableRows">
              <template #icon>
                <n-icon><TrashOutline/></n-icon>
              </template>
              {{ t("index.clear_list") }}
            </NButton>
          </template>
          <div>
            <div class="flex flex-row items-center text-red-700 my-2 text-base">
              <n-icon><TrashOutline/></n-icon>
              <p class="ml-1">{{ t("index.clear_list_tip") }}</p>
            </div>
            <NCheckbox v-model:checked="rememberChoiceTmp">
              <span class="text-gray-400">{{ t('index.remember_clear_choice') }}</span>
            </NCheckbox>
          </div>
        </n-popconfirm>

        <NPopover placement="bottom" trigger="hover">
          <template #trigger>
            <NButton quaternary type="info">
              <template #icon>
                <NIcon><Apps/></NIcon>
              </template>
              {{ t('index.more_operation') }}
            </NButton>
          </template>
          <div class="flex min-w-[150px] flex-col gap-1">
            <NButton tertiary type="warning" @click.stop="batchExport()">
              <template #icon>
                <n-icon><ArrowRedoCircleOutline/></n-icon>
              </template>
              {{ t('index.batch_export') }}
            </NButton>
            <NButton tertiary type="info" @click.stop="showImport=true">
              <template #icon>
                <n-icon><ServerOutline/></n-icon>
              </template>
              {{ t('index.batch_import') }}
            </NButton>
            <NButton tertiary type="primary" @click.stop="batchExport('url')">
              <template #icon>
                <n-icon><ArrowRedoCircleOutline/></n-icon>
              </template>
              {{ t('index.export_url') }}
            </NButton>
          </div>
        </NPopover>
      </div>
    </div>
    <div class="flex-1">
      <NDataTable
          :columns="columns"
          :data="filteredData"
          :bordered="false"
          :max-height="tableHeight"
          :row-key="rowKey"
          :virtual-scroll="true"
          :header-height="48"
          :height-for-row="()=> 48"
          :checked-row-keys="checkedRowKeysValue"
          @update:checked-row-keys="handleCheck"
          @update:filters="updateFilters"
          style="--wails-draggable:no-drag"
      />
    </div>
    <div class="flex justify-center items-center text-blue-400" id="bottom">
      <span class="cursor-pointer px-2 py-1" @click="BrowserOpenURL(certUrl)">{{ t('footer.cert_download') }}</span>
      <span class="cursor-pointer px-2 py-1" @click="BrowserOpenURL('https://github.com/putyy/res-downloader')">{{ t('footer.source_code') }}</span>
      <span class="cursor-pointer px-2 py-1" @click="BrowserOpenURL('https://github.com/putyy/res-downloader/issues')">{{ t('footer.help') }}</span>
      <span class="cursor-pointer px-2 py-1" @click="BrowserOpenURL('https://github.com/putyy/res-downloader/releases')">{{ t('footer.update_log') }}</span>
    </div>
    <Preview v-model:showModal="showPreviewRow" :previewRow="previewRow"/>
    <ShowLoading :loadingText="loadingText" :isLoading="loading"/>
    <ImportJson v-model:showModal="showImport" @submit="handleImport"/>
    <Password v-model:showModal="showPassword" @submit="handlePassword"/>
  </div>
</template>

<script lang="ts" setup>
import {NButton, NIcon, NImage, NInput, NSwitch, NTooltip, NPopover, NGradientText} from "naive-ui"
import {computed, h, onMounted, ref, watch} from "vue"
import type {appType} from "@/types/app"
import type {DataTableRowKey, ImageRenderToolbarProps, DataTableFilterState, DataTableBaseColumn} from "naive-ui"
import Preview from "@/components/Preview.vue"
import ShowLoading from "@/components/ShowLoading.vue"
// @ts-ignore
import {getDecryptionArray} from '@/assets/js/decrypt.js'
import {useIndexStore} from "@/stores"
import appApi from "@/api/app"
import Action from "@/components/Action.vue"
import ActionDesc from "@/components/ActionDesc.vue"
import ImportJson from "@/components/ImportJson.vue"
import {useEventStore} from "@/stores/event"
import {BrowserOpenURL, ClipboardSetText} from "../../wailsjs/runtime"
import Password from "@/components/Password.vue"
import ShowOrEdit from "@/components/ShowOrEdit.vue"
import {useI18n} from 'vue-i18n'
import {
  DownloadOutline,
  ArrowRedoCircleOutline,
  ServerOutline,
  SearchOutline,
  Apps,
  TrashOutline, CloseOutline,
  DocumentTextOutline,
  FolderOpenOutline,
  OpenOutline
} from "@vicons/ionicons5"
import {useDialog} from 'naive-ui'
import * as bind from "../../wailsjs/go/core/Bind"
import {Quit} from "../../wailsjs/runtime"
import {DialogOptions} from "naive-ui/es/dialog/src/DialogProvider"
import {formatSize} from "@/func"

const {t} = useI18n()
const eventStore = useEventStore()
const dialog = useDialog()
const isProxy = computed(() => {
  return store.isProxy
})
const certUrl = computed(() => {
  return store.baseUrl + "/api/cert"
})
const data = ref<any[]>([])
const filterClassify = ref<string[]>([])
const filteredData = computed(() => {
  let result = data.value

  if (filterClassify.value.length > 0) {
    result = result.filter(item => filterClassify.value.includes(item.Classify))
  }

  if (descriptionSearchValue.value) {
    result = result.filter(item => item.Description?.toLowerCase().includes(descriptionSearchValue.value.toLowerCase()))
  }

  if (urlSearchValue.value) {
    result = result.filter(item => item.Url?.toLowerCase().includes(urlSearchValue.value.toLowerCase()))
  }

  return result
})

const store = useIndexStore()
const tableHeight = ref(800)
const defaultResourceTypes = ["video", "m3u8"]
const resourcesType = ref<string[]>([...defaultResourceTypes])

const classifyAlias: { [key: string]: any } = {
  image: computed(() => t("index.image")),
  audio: computed(() => t("index.audio")),
  video: computed(() => t("index.video")),
  m3u8: computed(() => t("index.m3u8")),
  live: computed(() => t("index.live")),
  xls: computed(() => t("index.xls")),
  doc: computed(() => t("index.doc")),
  pdf: computed(() => t("index.pdf")),
  stream: computed(() => t("index.stream")),
  font: computed(() => t("index.font"))
}

const dwStatus = computed<any>(() => {
  return {
    ready: t("index.ready"),
    pending: t("index.pending"),
    running: t("index.running"),
    error: t("index.error"),
    done: t("index.done"),
    handle: t("index.handle"),
    extracting: t("index.extracting"),
    extract_done: t("index.extract_done"),
    extract_error: t("index.extract_error")
  }
})

const maxConcurrentDownloads = computed(() => {
  return store.globalConfig.DownNumber
})

const defaultClassifyTypes = ["video", "m3u8", "audio", "live", "image", "stream", "xls", "doc", "pdf", "font"]
const visibleClassifyTypes = ["video", "m3u8", "audio", "image"]
const buildClassifyOptions = (types: string[]) => [
  {value: "all", label: computed(() => t("index.all"))},
  ...types.map(type => ({
    value: type,
    label: classifyAlias[type] ?? type,
  })),
]

const classify = ref(buildClassifyOptions(defaultClassifyTypes))
const visibleClassify = computed(() => {
  return visibleClassifyTypes.reduce<any[]>((result, type) => {
    const item = classify.value.find(item => item.value === type)
    if (item) {
      result.push(item)
    }
    return result
  }, [])
})
const moreClassify = computed(() => {
  return classify.value.filter(item => item.value !== "all" && !visibleClassifyTypes.includes(item.value))
})
const selectedMoreClassifyCount = computed(() => {
  return moreClassify.value.filter(item => isResourceTypeActive(item.value)).length
})

const descriptionSearchValue = ref("")
const urlSearchValue = ref("")
const rememberChoice = ref(false)
const rememberChoiceTmp = ref(false)

const getClassifyLabel = (item: any) => {
  return item?.label?.value ?? item?.label ?? item?.value
}

const isResourceTypeActive = (type: string) => {
  return resourcesType.value.includes(type)
}

const normalizeResourceTypes = (types: unknown) => {
  const savedTypes = Array.isArray(types) ? types.filter((item): item is string => typeof item === "string") : []
  if (savedTypes.includes("all")) {
    return [...defaultResourceTypes]
  }
  return Array.from(new Set([...defaultResourceTypes, ...savedTypes]))
}

const updateResourceType = (type: string, checked: boolean) => {
  if (type === "all") {
    resourcesType.value = checked ? ["all"] : [...defaultResourceTypes]
    return
  }

  const next = resourcesType.value.filter(item => item !== "all" && item !== type)
  if (checked) {
    next.push(type)
  }
  resourcesType.value = next.length > 0 ? next : [...defaultResourceTypes]
}

const selectDownloadDir = () => {
  appApi.openDirectoryDialog().then((res: appType.Res) => {
    if (res.code === 1 && res.data?.folder) {
      store.setConfig({SaveDirectory: res.data.folder})
    }
  }).catch((err: any) => {
    window?.$message?.error(err?.message || String(err))
  })
}

const openSaveDirectory = () => {
  if (!store.globalConfig.SaveDirectory) {
    window?.$message?.error(t("index.save_path_empty"))
    return
  }
  appApi.openFolder({filePath: store.globalConfig.SaveDirectory})
}

const getDisplaySavePath = (row: appType.MediaInfo) => {
  if (row.Status === "extract_done" && row.CaptionPath) {
    return row.CaptionPath
  }
  return row.SavePath
}

const canOpenDisplaySavePath = (row: appType.MediaInfo) => {
  return ["done", "extract_done"].includes(row.Status) && !!getDisplaySavePath(row)
}

const columns = ref<any[]>([
  {
    type: "selection",
  },
  {
    title: () => {
      if (checkedRowKeysValue.value.length > 0) {
        return h(NGradientText, {type: "success"}, t("index.choice") + `(${checkedRowKeysValue.value.length})`)
      }
      return h('div', {class: 'flex items-center'}, [
        t('index.domain'),
        h(NPopover, {
          style: "--wails-draggable:no-drag",
          trigger: 'click',
          placement: 'bottom',
          showArrow: true,
        }, {
          trigger: () => h(NIcon, {
            size: "18",
            class: `ml-1 cursor-pointer ${urlSearchValue.value ? "text-green-600": "text-gray-500"}`,
            onClick: (e: MouseEvent) => e.stopPropagation()
          }, h(SearchOutline)),
          default: () => h('div', {class: 'p-2 w-64'}, [
            h(NInput, {
              value: urlSearchValue.value,
              'onUpdate:value': (val: string) => urlSearchValue.value = val,
              placeholder: t('index.search_description'),
              clearable: true
            }, {
              prefix: () => h(NIcon, {component: SearchOutline})
            })
          ])
        })
      ])
    },
    key: "Domain",
    width: 90,
    render: (row: appType.MediaInfo) => {
      return h(NTooltip, {
        trigger: 'hover',
        placement: 'top'
      }, {
        trigger: () => h('span', {
          class: 'cursor-default'
        }, row.Domain),
        default: () => row.Url
      })
    }
  },
  {
    title: computed(() => t("index.type")),
    key: "Classify",
    width: 80,
    filterOptions: computed(() => Array.from(classify.value).slice(1)),
    filterMultiple: true,
    filter: (value: string, row: appType.MediaInfo) => {
      return !!~row.Classify.indexOf(String(value))
    },
    render: (row: appType.MediaInfo) => {
      const item = classify.value.find(item => item.value === row.Classify)
      return item ? item.label : row.Classify
    }
  },
  {
    title: computed(() => t("index.preview")),
    key: "Url",
    width: 80,
    render: (row: appType.MediaInfo) => {
      if (row.Classify === "image") {
        return h("div", {
          style: "width: 100%;max-height:80px;overflow:hidden;"
        }, h(NImage, {
          objectFit: "contain",
          lazy: true,
          "render-toolbar": renderToolbar,
          src: row.Url
        }))
      }
      return [
        h(
            NButton,
            {
              strong: true,
              tertiary: true,
              type: "info",
              size: "small",
              style: {
                margin: "2px"
              },
              onClick: () => {
                if (row.Classify === "audio" || row.Classify === "video" || row.Classify === "m3u8" || row.Classify === "live") {
                  previewRow.value = row
                  showPreviewRow.value = true
                }
              }
            },
            {
              default: () => {
                if (row.Classify === "audio" || row.Classify === "video" || row.Classify === "m3u8" || row.Classify === "live") {
                  return t("index.preview")
                }
                return t("index.preview_tip")
              }
            }
        ),
      ]
    }
  },
  {
    title: computed(() => t("index.status")),
    key: "Status",
    width: 110,
    render: (row: appType.MediaInfo, index: number) => {
      let status = "info"
      if (row.Status === "done" || row.Status === "running") {
        status = "success"
      } else if (row.Status === "pending") {
        status = "warning"
      } else if (row.Status === "extracting") {
        status = "warning"
      } else if (row.Status === "extract_done") {
        status = "success"
      } else if (row.Status === "extract_error") {
        status = "error"
      }

      return h(
          NButton,
          {
            tertiary: true,
            type: status as any,
            size: "small",
            style: {
              margin: "2px",
              color: row.Status === "extract_error" ? "#d03050" : undefined
            },
            onClick: () => {
              if (canOpenDisplaySavePath(row)) {
                appApi.openFolder({filePath: getDisplaySavePath(row)})
              } else if (row.Status === "ready") {
                download(row, index)
              }
            }
          },
          {
            default: () => {
              return row.Status === "running" ? row.SavePath : dwStatus.value[row.Status as keyof typeof dwStatus]
            }
          }
      )
    }
  },
  {
    title: () => h('div', {class: 'flex items-center'}, [
      t('index.description'),
      h(NPopover, {
        style: "--wails-draggable:no-drag",
        trigger: 'click',
        placement: 'bottom',
        showArrow: true,
      }, {
        trigger: () => h(NIcon, {
          size: "18",
          class: `ml-1 cursor-pointer ${descriptionSearchValue.value ? "text-green-600": "text-gray-500"}`,
          onClick: (e: MouseEvent) => e.stopPropagation()
        }, h(SearchOutline)),
        default: () => h('div', {class: 'p-2 w-64'}, [
          h(NInput, {
            value: descriptionSearchValue.value,
            'onUpdate:value': (val: string) => descriptionSearchValue.value = val,
            placeholder: t('index.search_description'),
            clearable: true
          }, {
            prefix: () => h(NIcon, {component: SearchOutline})
          })
        ])
      })
    ]),
    key: "Description",
    width: 150,
    render: (row: appType.MediaInfo, index: number) => {
      return h(ShowOrEdit, {
        value: row.Description,
        onUpdateValue(v: string) {
          data.value[index].Description = v
          cacheData()
        }
      })
    }
  },
  {
    title: computed(() => t("index.resource_size")),
    key: "Size",
    width: 120,
    sorter: (row1: appType.MediaInfo, row2: appType.MediaInfo) => row1.Size - row2.Size,
    render(row: appType.MediaInfo, index: number) {
      return formatSize(row.Size)
    }
  },
  {
    title: computed(() => t("index.save_path")),
    key: "SavePath",
    render(row: appType.MediaInfo, index: number) {
      return h("a",
          {
            href: "javascript:;",
            class: "ellipsis-2",
            style: {
              color: "#5a95d0"
            },
            onClick: () => {
              if (canOpenDisplaySavePath(row)) {
                appApi.openFolder({filePath: getDisplaySavePath(row)})
              }
            }
          },
          row.Status === "running" ? "" : getDisplaySavePath(row)
      )
    }
  },
  {
    key: "actions",
    width: 160,
    render(row: appType.MediaInfo, index: number) {
      return h(Action, {key: index, row: row, index: index, onAction: dataAction})
    },
    title() {
      return h(ActionDesc)
    }
  }
])

const checkedRowKeysValue = ref<DataTableRowKey[]>([])
const showPreviewRow = ref(false)
const previewRow = ref<appType.MediaInfo>()
const loading = ref(false)
const loadingText = ref("")
const showImport = ref(false)
const showPassword = ref(false)
const downloadQueue = ref<appType.MediaInfo[]>([])
let activeDownloads = 0
let isOpenProxy = false
let isInstall = false

onMounted(() => {
  try {
    window.addEventListener("resize", () => {
      resetTableHeight()
    })
    loading.value = true
    handleInstall().then((is: boolean) => {
      isInstall = true
      loading.value = false
    })

    checkLoading()
    watch(showPassword, () => {
      if (!showPassword.value) {
        checkLoading()
      }
    })
  } catch (e) {
    window.$message?.error(JSON.stringify(e), {duration: 5000})
  }

  buildClassify()

  const temp = localStorage.getItem("resources-type")
  if (temp) {
    resourcesType.value = normalizeResourceTypes(JSON.parse(temp).res)
  } else {
    appApi.setType(resourcesType.value)
  }

  const cache = localStorage.getItem("resources-data")
  if (cache) {
    data.value = JSON.parse(cache)
  }

  const choiceCache = localStorage.getItem("remember-clear-choice")
  if (choiceCache === "1") {
    rememberChoice.value = true
  }

  watch(rememberChoice, (n, o) => {
    if (rememberChoice.value) {
      localStorage.setItem("remember-clear-choice", "1")
    } else {
      localStorage.removeItem("remember-clear-choice")
    }
  })

  resetTableHeight()

  eventStore.addHandle({
    type: "newResources",
    event: (res: appType.MediaInfo) => {
      if (store.globalConfig.InsertTail) {
        data.value.push(res)
      } else {
        data.value.unshift(res)
      }
      cacheData()
    }
  })

  eventStore.addHandle({
    type: "downloadProgress",
    event: (res: { Id: string, SavePath: string, Status: string, Message: string }) => {
      switch (res.Status) {
        case "running":
          updateItem(res.Id, item => {
            item.SavePath = res.Message
            item.Status = 'running'
          })
          break
        case "done":
          updateItem(res.Id, item => {
            item.SavePath = res.SavePath
            item.Status = 'done'
          })
          if (activeDownloads > 0) {
            activeDownloads--
          }
          cacheData()
          checkQueue()
          // 检查是否有等待提取文案的任务
          if (pendingCaptionExtractions.value.has(res.Id)) {
            pendingCaptionExtractions.value.delete(res.Id)
            const idx = data.value.findIndex(item => item.Id === res.Id)
            if (idx !== -1) {
              doExtractCaption(data.value[idx], idx)
            }
          }
          break
        case "error":
          updateItem(res.Id, item => {
            item.SavePath = res.Message
            item.Status = 'error'
          })
          if (activeDownloads > 0) {
            activeDownloads--
          }
          cacheData()
          checkQueue()
          // 下载失败，取消等待提取文案
          if (pendingCaptionExtractions.value.has(res.Id)) {
            pendingCaptionExtractions.value.delete(res.Id)
            window?.$message?.error(t("index.extract_caption_error"))
          }
          break
      }
    }
  })
})

watch(() => {
  return store.globalConfig.MimeMap
}, () => {
  buildClassify()
})

watch(resourcesType, (n, o) => {
  localStorage.setItem("resources-type", JSON.stringify({res: resourcesType.value}))
  appApi.setType(resourcesType.value)
})

const updateItem = (id: string, updater: (item: any) => void) => {
  const item = data.value.find(i => i.Id === id)
  if (item) updater(item)
}

function cacheData() {
  localStorage.setItem("resources-data", JSON.stringify(data.value))
}

const resetTableHeight = () => {
  try {
    const headerHeight = document.getElementById("header")?.offsetHeight || 0
    const bottomHeight = document.getElementById("bottom")?.offsetHeight || 0
    // @ts-ignore
    const theadHeight = document.getElementsByClassName("n-data-table-thead")[0]?.offsetHeight || 0
    const height = document.documentElement.clientHeight || window.innerHeight
    tableHeight.value = height - headerHeight - bottomHeight - theadHeight - 20
  } catch (e) {
    console.log(e)
  }
}

const buildClassify = () => {
  const mimeMap = store.globalConfig.MimeMap ?? {}
  const seen = new Set()
  // 常用类型排序优先级（越小越靠前）
  const typePriority: { [key: string]: number } = {
    video: 1,
    m3u8: 2,
    audio: 3,
    live: 4,
    image: 5,
  }
  const types = Object.values(mimeMap)
      .filter(({Type}) => {
        if (seen.has(Type)) return false
        seen.add(Type)
        return true
      })
      .map(({Type}) => Type)
      .sort((a, b) => {
        const pa = typePriority[a] ?? 100
        const pb = typePriority[b] ?? 100
        return pa - pb
      })
  classify.value = buildClassifyOptions(types.length > 0 ? types : defaultClassifyTypes)
}

const dataAction = (row: appType.MediaInfo, index: number, type: string) => {
  switch (type) {
    case "down":
      download(row, index)
      break
    case "cancel":
      if (row.Status === "pending") {
        const queueIndex = downloadQueue.value.findIndex(item => item.Id === row.Id)
        if (queueIndex !== -1) {
          downloadQueue.value.splice(queueIndex, 1)
        }
        updateItem(row.Id, item => {
          item.Status = 'ready'
          item.SavePath = ''
        })
        cacheData()
      } else if (row.Status === "running") {
        appApi.cancel({id: row.Id}).then((res) => {
          updateItem(row.Id, item => {
            item.Status = 'ready'
            item.SavePath = ''
          })
          if (activeDownloads > 0) {
            activeDownloads--
          }
          cacheData()
          checkQueue()
          if (res.code === 0) {
            window?.$message?.error(res.message)
            return
          }
        })
      }
      break
    case "copy":
      ClipboardSetText(row.Url).then((is: boolean) => {
        if (is) {
          window?.$message?.success(t("common.copy_success"))
        } else {
          window?.$message?.error(t("common.copy_fail"))
        }
      })
      break
    case "json":
      ClipboardSetText(encodeURIComponent(JSON.stringify(row))).then((is: boolean) => {
        if (is) {
          window?.$message?.success(t("common.copy_success"))
        } else {
          window?.$message?.error(t("common.copy_fail"))
        }
      })
      break
    case "open":
      BrowserOpenURL(row.Url)
      break
    case "decode":
      decodeWxFile(row, index)
      break
    case "extract_caption":
      extractCaption(row, index)
      break
    case "delete":
      if (row.Status === "pending" || row.Status === "running") {
        window?.$message?.error(t("index.delete_tip"))
        return
      }
      appApi.delete({sign: [row.UrlSign]}).then(() => {
        data.value.splice(index, 1)
        cacheData()
      })
      break
  }
}

const renderToolbar = ({nodes}: ImageRenderToolbarProps) => {
  return [
    nodes.rotateCounterclockwise,
    nodes.rotateClockwise,
    nodes.resizeToOriginalSize,
    nodes.zoomOut,
    nodes.zoomIn,
    nodes.close
  ]
}

const rowKey = (row: appType.MediaInfo) => {
  return row.Id
}

const handleCheck = (rowKeys: DataTableRowKey[]) => {
  checkedRowKeysValue.value = rowKeys
}

const updateFilters = (filters: DataTableFilterState, initiatorColumn: DataTableBaseColumn) => {
  filterClassify.value = filters.Classify as string[]
}

const batchDown = async () => {
  if (checkedRowKeysValue.value.length <= 0) {
    window?.$message?.error(t("index.use_data"))
    return
  }

  if (!store.globalConfig.SaveDirectory) {
    window?.$message?.error(t("index.save_path_empty"))
    return
  }

  data.value.forEach((item, index) => {
    if (checkedRowKeysValue.value.includes(item.Id) && item.Classify !== 'live' && item.Classify !== 'm3u8') {
      download(item, index)
    }
  })

  checkedRowKeysValue.value = []
}

const batchExtractCaption = () => {
  if (checkedRowKeysValue.value.length <= 0) {
    window?.$message?.error(t("index.use_data"))
    return
  }

  // 筛选出选中的视频项
  const videoItems: { row: appType.MediaInfo, index: number }[] = []
  data.value.forEach((item, index) => {
    if (checkedRowKeysValue.value.includes(item.Id) && item.Classify === 'video') {
      videoItems.push({ row: item, index })
    }
  })

  if (videoItems.length === 0) {
    window?.$message?.warning(t("index.batch_extract_caption_no_video"))
    return
  }

  // 分成已下载和未下载两组
  const isVideoDownloaded = (v: { row: appType.MediaInfo }) => 
    v.row.SavePath && (v.row.Status === 'done' || v.row.Status === 'extract_done' || v.row.Status === 'extract_error')
  const downloaded = videoItems.filter(v => isVideoDownloaded(v))
  const notDownloaded = videoItems.filter(v => !isVideoDownloaded(v))

  // 未下载的先触发下载，下载完后自动提取
  notDownloaded.forEach(({ row, index }) => {
    pendingCaptionExtractions.value.set(row.Id, index)
    if (row.Status !== 'running' && row.Status !== 'pending') {
      download(row, index)
    }
  })

  if (notDownloaded.length > 0) {
    window?.$message?.info(t("index.batch_extract_caption_downloading", { count: notDownloaded.length }))
  }

  // 已下载的加入提取队列
  downloaded.forEach(({ row, index }) => {
    doExtractCaption(row, index)
  })

  if (downloaded.length > 0) {
    window?.$message?.info(t("index.batch_extract_caption_started", { count: downloaded.length }))
  }

  checkedRowKeysValue.value = []
}

const batchCancel = async () => {
  checkedRowKeysValue.value = data.value
      .filter(item => item.Status === "pending" || item.Status === "running")
      .map(item => item.Id)

  if (checkedRowKeysValue.value.length <= 0) {
    window?.$message?.error(t("index.use_data"))
    return
  }

  loading.value = true
  const cancelKeys = new Set(checkedRowKeysValue.value)
  const cancelTasks: Promise<any>[] = []
  data.value.forEach((item) => {
    if (!cancelKeys.has(item.Id)) {
      return
    }

    if (item.Status === "pending") {
      const queueIndex = downloadQueue.value.findIndex(qItem => qItem.Id === item.Id)
      if (queueIndex !== -1) {
        downloadQueue.value.splice(queueIndex, 1)
      }
      item.Status = 'ready'
      item.SavePath = ''
      return
    }

    if (item.Status === "running") {
      if (activeDownloads > 0) {
        activeDownloads--
      }
      cancelTasks.push(appApi.cancel({id: item.Id}).then(() => {
        item.Status = 'ready'
        item.SavePath = ''
        checkQueue()
      }))
    }
  })
  await Promise.allSettled(cancelTasks)
  loading.value = false
  checkedRowKeysValue.value = []
  cacheData()
}

const batchExport = (type?: string) => {
  if (checkedRowKeysValue.value.length <= 0) {
    window?.$message?.error(t("index.use_data"))
    return
  }

  if (!store.globalConfig.SaveDirectory) {
    window?.$message?.error(t("index.save_path_empty"))
    return
  }

  loadingText.value = t("common.loading")
  loading.value = true

  let jsonData = data.value.filter(item => checkedRowKeysValue.value.includes(item.Id))

  if (type === "url") {
    jsonData = jsonData.map(item => item.Url)
  } else {
    jsonData = jsonData.map(item => encodeURIComponent(JSON.stringify(item)))
  }

  appApi.batchExport({content: jsonData.join("\n")}).then((res: appType.Res) => {
    loading.value = false
    if (res.code === 0) {
      window?.$message?.error(res.message)
      return
    }
    window?.$message?.success(t("index.import_success"))
    window?.$message?.info(t("index.save_path") + "：" + res.data?.file_name, {
      duration: 5000
    })
  })
}

const uint8ArrayToBase64 = (bytes: any) => {
  return window.btoa(Array.from(bytes, (byte: any) => String.fromCharCode(byte)).join(''))
}

const download = (row: appType.MediaInfo, index: number) => {
  if (!store.globalConfig.SaveDirectory) {
    window?.$message?.error(t("index.save_path_empty"))
    return
  }

  if (data.value.some(item => item.Id === row.Id && item.Status === "running")) {
    return
  }

  if (downloadQueue.value.some(item => item.Id === row.Id || item.Url === row.Url)) {
    return
  }

  if (activeDownloads >= maxConcurrentDownloads.value) {
    row.Status = "pending"
    downloadQueue.value.push(row)
    window?.$message?.info(t("index.download_queued", {count: downloadQueue.value.length}))
    return
  }

  startDownload(row, index)
}

const startDownload = (row: appType.MediaInfo, index: number) => {
  activeDownloads++

  const decodeStr = row.DecodeKey
      ? uint8ArrayToBase64(getDecryptionArray(row.DecodeKey))
      : ""

  appApi.download({...row, decodeStr}).then((res: appType.Res) => {
    if (res.code === 0) {
      window?.$message?.error(res.message)
    }
  })
}

const checkQueue = () => {
  if (downloadQueue.value.length > 0 && activeDownloads < maxConcurrentDownloads.value) {
    const nextItem = downloadQueue.value.shift()
    if (nextItem) {
      const index = data.value.findIndex(item => item.Id === nextItem.Id)
      if (index !== -1) {
        startDownload(nextItem, index)
      }
    }
  }
}

const open = () => {
  isOpenProxy = true
  store.openProxy().then((res: appType.Res) => {
    if (res.code === 1) {
      return
    }

    if (["darwin", "linux"].includes(store.envInfo.platform)) {
      showPassword.value = true
    } else {
      window.$message?.error(res.message)
    }
  })
}

const close = () => {
  store.unsetProxy()
}

const toggleProxy = (checked: boolean) => {
  if (checked) {
    open()
    return
  }
  close()
}

const getClearableRowKeys = () => {
  return data.value
      .filter(item => item.Status !== "pending" && item.Status !== "running")
      .map(item => item.Id)
}

const selectClearableRows = () => {
  checkedRowKeysValue.value = getClearableRowKeys()
}

const clear = async () => {
  selectClearableRows()
  const clearKeys = new Set(checkedRowKeysValue.value)
  const newData = [] as any[]
  const signs: string[] = []

  data.value.forEach((item) => {
    if (clearKeys.has(item.Id) && item.Status !== "pending" && item.Status !== "running") {
      signs.push(item.UrlSign)
      return
    }
    newData.push(item)
  })

  checkedRowKeysValue.value = []
  await appApi.delete({sign: signs})
  data.value = newData
  cacheData()
}

const decodeWxFile = (row: appType.MediaInfo, index: number) => {
  if (!row.DecodeKey) {
    window?.$message?.error(t("index.video_decode_no"))
    return
  }
  appApi.openFileDialog().then((res: appType.Res) => {
    if (res.code === 0) {
      window?.$message?.error(res.message)
      return
    }
    if (res.data.file) {
      loadingText.value = t("index.video_decode_loading")
      loading.value = true
      appApi.wxFileDecode({
        ...row,
        filename: res.data.file,
        decodeStr: uint8ArrayToBase64(getDecryptionArray(row.DecodeKey))
      }).then((res: appType.Res) => {
        loading.value = false
        if (res.code === 0) {
          window?.$message?.error(res.message)
          return
        }
        data.value[index].SavePath = res.data.save_path
        data.value[index].Status = "done"
        cacheData()
        window?.$message?.success(t("index.video_decode_success"))
      })
    }
  })
}

// 下载完成后需要自动提取文案的项目 (Id -> index)
const pendingCaptionExtractions = ref<Map<string, number>>(new Map())

const extractCaption = (row: appType.MediaInfo, index: number) => {
  // 如果视频还没下载，先自动下载
  const isDownloaded = row.SavePath && (row.Status === 'done' || row.Status === 'extract_done' || row.Status === 'extract_error')
  if (!isDownloaded) {
    if (row.Status === 'running' || row.Status === 'pending') {
      // 已在下载中，标记等待提取
      pendingCaptionExtractions.value.set(row.Id, index)
      window?.$message?.info(t("index.extract_caption_downloading"))
      return
    }
    // 标记等待提取，然后触发下载
    pendingCaptionExtractions.value.set(row.Id, index)
    window?.$message?.info(t("index.extract_caption_auto_download"))
    download(row, index)
    return
  }
  
  doExtractCaption(row, index)
}

// 提取文案队列（最多5个并发）
const captionQueue = ref<{ row: appType.MediaInfo, index: number }[]>([])
let activeCaptions = 0
const maxConcurrentCaptions = 5

const processOneCaptionTask = async (row: appType.MediaInfo, index: number) => {
  // 根据 Id 查找实际 index（避免列表刷新导致 index 对不上）
  const realIndex = data.value.findIndex(item => item.Id === row.Id)
  if (realIndex === -1) {
    console.warn('提取文案: 找不到项目', row.Id)
    activeCaptions--
    processCaptionQueue()
    return
  }
  const item = data.value[realIndex]

  item.Status = 'extracting'
  try {
    const res: appType.Res = await appApi.extractCaption(item)
    console.log('提取文案响应:', res)
    // 后端成功返回 code=1，失败返回 code=0
    if (res.code === 1) {
      item.Status = 'extract_done'
      const txtPath = res.data?.txt_path || ''
      item.CaptionPath = txtPath
      if (res.data?.deleted_video) {
        item.SavePath = ''
      }
      const cleanupErrors = res.data?.cleanup_errors || []
      if (txtPath) {
        window?.$message?.success(t("index.extract_caption_saved") + `: ${txtPath}`)
      } else {
        window?.$message?.success(t("index.extract_caption_saved"))
      }
      if (cleanupErrors.length > 0) {
        window?.$message?.warning(t("index.extract_caption_cleanup_error", { errors: cleanupErrors.join('; ') }))
      }
    } else {
      item.Status = 'extract_error'
      const tempAudioPath = res.data?.temp_audio_path || ''
      if (tempAudioPath) {
        item.SavePath = tempAudioPath
      } else if (res.data?.deleted_video) {
        item.SavePath = ''
      }
      const cleanupErrors = res.data?.cleanup_errors || []
      const errMsg = res.message || t("index.extract_caption_error")
      window?.$message?.error(`${item.Domain || '提取文案'}: ${errMsg}`)
      if (cleanupErrors.length > 0) {
        window?.$message?.warning(t("index.extract_caption_cleanup_error", { errors: cleanupErrors.join('; ') }))
      }
      console.error('提取文案失败:', errMsg)
    }
  } catch (err: any) {
    item.Status = 'extract_error'
    const errMsg = err?.message || t("index.extract_caption_error")
    window?.$message?.error(`${item.Domain || '提取文案'}: ${errMsg}`)
    console.error('提取文案异常:', err)
  }
  cacheData()

  activeCaptions--
  processCaptionQueue()
}

const processCaptionQueue = () => {
  while (captionQueue.value.length > 0 && activeCaptions < maxConcurrentCaptions) {
    const task = captionQueue.value.shift()!
    activeCaptions++
    processOneCaptionTask(task.row, task.index)
  }
}

const doExtractCaption = (row: appType.MediaInfo, index: number) => {
  const realIndex = data.value.findIndex(item => item.Id === row.Id)
  if (realIndex === -1) return
  const item = data.value[realIndex]
  // 避免重复加入队列
  if (item.Status === 'extracting' || captionQueue.value.some(task => task.row.Id === item.Id)) return
  item.Status = 'extracting'
  captionQueue.value.push({ row: item, index: realIndex })
  cacheData()
  processCaptionQueue()
}

const handleImport = (content: string) => {
  if (!content) {
    window?.$message?.error(t("view.import_empty"))
    return
  }
  let newItems = [] as any[]
  content.split("\n").forEach((line, index) => {
    try {
      let res = JSON.parse(decodeURIComponent(line))
      if (res && res?.Id) {
        res.Id = res.Id + Math.floor(Math.random() * 100000)
        res.SavePath = ""
        res.Status = "ready"
        newItems.push(res)
      }
    } catch (e) {
      console.log(e)
    }
  })
  if (newItems.length > 0) {
    data.value = [...newItems, ...data.value]
    cacheData()
  }
  showImport.value = false
}

const handlePassword = async (password: string, isCache: boolean) => {
  const res = await appApi.setSystemPassword({password, isCache})
  if (res.code === 0) {
    window.$message?.error(res.message)
    return
  }

  if (isOpenProxy) {
    showPassword.value = false
    store.openProxy()
    return
  }

  handleInstall().then((is: boolean) => {
    if (is) {
      showPassword.value = false
    }
  })
}

const handleInstall = async () => {
  isOpenProxy = false
  const res = await appApi.install()
  if (res.code === 1) {
    store.globalConfig.AutoProxy && store.openProxy()
    return true
  }

  window.$message?.error(res.message, {duration: 5000})

  if (store.envInfo.platform === "windows" && res.message.includes("Access is denied")) {
    window.$message?.error(t("index.win_install_tip"))
  } else if (["darwin", "linux"].includes(store.envInfo.platform)) {
    showPassword.value = true
  }
  return false
}

const checkLoading = () => {
  setTimeout(() => {
    if (loading.value && !isInstall && !showPassword.value) {
      dialog.warning({
        title: t("index.start_err_tip"),
        content: t("index.start_err_content"),
        positiveText: t("index.start_err_positiveText"),
        negativeText: t("index.start_err_negativeText"),
        draggable: false,
        closeOnEsc: false,
        closable: false,
        maskClosable: false,
        onPositiveClick: () => {
          bind.ResetApp()
        },
        onNegativeClick: () => {
          Quit()
        }
      } as DialogOptions)
    }
  }, 6000)
}
</script>
