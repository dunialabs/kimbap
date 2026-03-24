import { Switch } from '@/components/ui/switch'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface PublicAccessConfigurationProps {
  checked: boolean
  onCheckedChange: (checked: boolean) => void
}

export function PublicAccessConfiguration({
  checked,
  onCheckedChange
}: PublicAccessConfigurationProps) {
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white">Public Access</h3>
      </div>
      <p className="text-xs text-gray-500 dark:text-gray-400">
        When enabled, all users can access this server unless restricted by their permission list.
      </p>
      <div className="border rounded-lg p-3 bg-gray-50 dark:bg-gray-900">
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                    Enable Public Access
                  </span>
                </div>
                <Switch
                  checked={checked}
                  onCheckedChange={onCheckedChange}
                  size="sm"
                />
              </div>
            </TooltipTrigger>
            <TooltipContent>
              <p className="text-sm max-w-xs">Allow unrestricted access for all users by default.</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
    </div>
  )
}
