import type { FileNode } from '../../types'
import { isDesktopApp, desktopSaveBlob } from '../../utils/desktop-download'

/** Trigger a file download from a Blob. Uses native save dialog on desktop. */
export async function downloadBlob(blob: Blob, filename: string): Promise<void> {
  if (await isDesktopApp()) {
    await desktopSaveBlob(blob, filename)
    return
  }
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

/** Recursively filter a FileNode tree by name substring match. */
export function filterTree(node: FileNode, query: string): FileNode | null {
  if (node.name.toLowerCase().includes(query)) {
    return node
  }

  if (node.type === 'dir' && node.children) {
    const filteredChildren = node.children
      .map((child) => filterTree(child, query))
      .filter((child): child is FileNode => child !== null)

    if (filteredChildren.length > 0) {
      return { ...node, children: filteredChildren }
    }
  }

  return null
}
