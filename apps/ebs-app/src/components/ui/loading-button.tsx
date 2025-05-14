import { Button } from "@/components/ui/button";
import { Loader } from "lucide-react";
import { ComponentProps } from "react";

const LoadingButton = ({ text = 'Loading', icon = true, className, variant = 'secondary' }: { text?: string, icon?: boolean, variant?: 'link' | 'default' | 'secondary' | 'ghost' | 'destructive' | 'outline' } & ComponentProps<'button'>) => {
  return (
    <div className="flex items-center gap-2">
      {icon ?
      <Button size="icon" variant="secondary" className={className}>
        <Loader className="animate-spin" />
      </Button> :
      <Button variant={variant} className={className}>
        <Loader className="animate-spin" /> {text}
      </Button>
      }
    </div>
  );
};

export default LoadingButton;
