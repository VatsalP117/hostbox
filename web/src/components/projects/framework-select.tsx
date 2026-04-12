import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { frameworkConfig } from "@/lib/constants";
import type { Framework } from "@/types/models";

interface FrameworkSelectProps {
  value: Framework | "";
  onChange: (value: Framework | "") => void;
}

export function FrameworkSelect({ value, onChange }: FrameworkSelectProps) {
  return (
    <Select
      value={value || "auto"}
      onValueChange={(v) => onChange(v === "auto" ? "" : (v as Framework))}
    >
      <SelectTrigger>
        <SelectValue placeholder="Auto-detect" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="auto">Auto-detect</SelectItem>
        {(Object.entries(frameworkConfig) as [Framework, { label: string }][]).map(
          ([key, config]) => (
            <SelectItem key={key} value={key}>
              {config.label}
            </SelectItem>
          ),
        )}
      </SelectContent>
    </Select>
  );
}
