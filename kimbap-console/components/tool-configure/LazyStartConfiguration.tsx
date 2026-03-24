import { Switch } from '@/components/ui/switch';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';

interface LazyStartConfigurationProps {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  supportsIdleSleep?: boolean;
}

export function LazyStartConfiguration({
  checked,
  onCheckedChange,
  supportsIdleSleep = true,
}: LazyStartConfigurationProps) {
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white">Lazy Start</h3>
      </div>
      <p className="text-xs text-gray-500 dark:text-gray-400">
        {supportsIdleSleep
          ? 'Enable lazy loading for this server. When enabled, the server delays startup until first use and automatically shuts down when idle.'
          : 'Enable lazy loading for this server. When enabled, the server delays startup until first use.'}
      </p>
      <div className="border rounded-lg p-3 bg-gray-50 dark:bg-gray-900">
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                    Enable Lazy Start
                  </span>
                </div>
                <Switch checked={checked} onCheckedChange={onCheckedChange} size="sm" />
              </div>
            </TooltipTrigger>
            <TooltipContent>
              <p className="text-sm max-w-xs">
                {supportsIdleSleep
                  ? 'Delay startup until first use and auto-stop when idle.'
                  : 'Delay startup until first use.'}
              </p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
    </div>
  );
}
