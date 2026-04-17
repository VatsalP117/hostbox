import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import type { NotificationEvent } from "@/types/models";

const allEvents: { value: NotificationEvent; label: string }[] = [
  { value: "deploy_failure", label: "Deployment Failed" },
  { value: "deploy_success", label: "Deployment Succeeded" },
  { value: "domain_verified", label: "Domain Verified" },
];

interface EventToggleListProps {
  selected: NotificationEvent[];
  onChange: (events: NotificationEvent[]) => void;
}

export function EventToggleList({ selected, onChange }: EventToggleListProps) {
  const toggle = (event: NotificationEvent) => {
    if (selected.includes(event)) {
      onChange(selected.filter((e) => e !== event));
    } else {
      onChange([...selected, event]);
    }
  };

  return (
    <div className="space-y-2">
      {allEvents.map((event) => (
        <div key={event.value} className="flex items-center gap-2">
          <Checkbox
            id={event.value}
            checked={selected.includes(event.value)}
            onCheckedChange={() => toggle(event.value)}
          />
          <Label htmlFor={event.value} className="text-sm font-normal">
            {event.label}
          </Label>
        </div>
      ))}
    </div>
  );
}
