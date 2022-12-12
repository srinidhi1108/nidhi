import { ReactNode } from 'react';

type DropdownProps = {
  isOpen: boolean;
  toggle: () => void;
  children: ReactNode;
};

function Dropdown({ isOpen, toggle, children }: DropdownProps) {
  return (
    <button
      className={`flex items-center font-medium text-sm rounded-lg h-[2.5rem] px-3 gap-2 text-primary border-2 border-primary active:border-primary active:text-primary transition-colors ${
        isOpen
          ? 'bg-komiser-200/50'
          : 'bg-transparent hover:bg-komiser-200/30 active:bg-komiser-200'
      }`}
      onClick={toggle}
    >
      {children}
    </button>
  );
}

export default Dropdown;
