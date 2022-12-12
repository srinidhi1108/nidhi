import Button from '../../../button/Button';
import inventoryFilterFieldOptions from './InventoryFilterFieldOptions';

type InventoryFilterFieldProps = {
  handleField: (field: string) => void;
};

function InventoryFilterField({ handleField }: InventoryFilterFieldProps) {
  return (
    <>
      {inventoryFilterFieldOptions.map((option, idx) => (
        <Button
          key={idx}
          size="sm"
          style="ghost"
          align="left"
          gap="md"
          transition={false}
          onClick={() => handleField(option.value)}
        >
          {option.icon}
          {option.label}
        </Button>
      ))}
    </>
  );
}

export default InventoryFilterField;
