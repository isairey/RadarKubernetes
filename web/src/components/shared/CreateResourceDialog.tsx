import { type ComponentProps } from 'react'
import { CreateResourceDialog as BaseCreateResourceDialog } from '@skyhook-io/k8s-ui'
import { useApplyResource } from '../../api/client'

type BaseProps = ComponentProps<typeof BaseCreateResourceDialog>

export function CreateResourceDialog(props: Omit<BaseProps, 'onApply' | 'isApplying'>) {
  const applyResource = useApplyResource()

  return (
    <BaseCreateResourceDialog
      {...props}
      onApply={(params) => applyResource.mutateAsync(params)}
      isApplying={applyResource.isPending}
    />
  )
}
