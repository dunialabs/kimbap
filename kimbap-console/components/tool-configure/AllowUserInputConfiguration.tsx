import { Switch } from '@/components/ui/switch';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';

interface AllowUserInputConfigurationProps {
  checked: number;
  onCheckedChange: (checked: number) => void;
}

export function AllowUserInputConfiguration({
  checked,
  onCheckedChange,
}: AllowUserInputConfigurationProps) {
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white">User Credentials</h3>
      </div>
      <p className="text-xs text-gray-500 dark:text-gray-400">
        When enabled, users must provide their own credentials (API keys, tokens, etc.) to use this
        tool via Kimbap Desk.
      </p>
      <div className="border rounded-lg p-3 bg-gray-50 dark:bg-gray-900">
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                    Enable User Credentials
                  </span>
                </div>
                <Switch
                  checked={checked === 1}
                  onCheckedChange={(value: boolean) => onCheckedChange(value ? 1 : 0)}
                  size="sm"
                />
              </div>
            </TooltipTrigger>
            <TooltipContent>
              <p className="text-sm max-w-xs">Require each user to supply their own credentials.</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
    </div>
  );
}
