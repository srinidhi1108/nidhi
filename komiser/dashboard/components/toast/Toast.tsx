import Button from '../button/Button';
import { ToastProps } from './hooks/useToast';

type ToastProp = ToastProps & {
  dismissToast: () => void;
};

function Toast({ hasError, title, message, dismissToast }: ToastProp) {
  return (
    <>
      <div
        className={`fixed overflow-hidden opacity-0 flex items-center justify-between max-w-2xl z-40 py-4 px-6 bottom-4 left-4 right-4 sm:left-8 rounded-lg shadow-2xl text-black-900 animate-fade-in-up ${
          hasError ? `bg-error-100 ` : `bg-success-100 `
        }`}
      >
        <div
          className={`absolute h-1 bottom-0 left-0 animate-width-to-fit ${
            hasError ? `bg-error-600/60 ` : `bg-success-600/60 `
          }`}
        ></div>
        <div className="flex items-center gap-4">
          <div
            className={`${hasError ? `text-error-600` : `text-success-600`}`}
          >
            {hasError ? (
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="24"
                height="24"
                fill="none"
                viewBox="0 0 24 24"
              >
                <path
                  stroke="currentColor"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth="2"
                  d="M12 22c5.5 0 10-4.5 10-10S17.5 2 12 2 2 6.5 2 12s4.5 10 10 10zM12 8v5M11.995 16h.009"
                ></path>
              </svg>
            ) : (
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="24"
                height="24"
                fill="none"
                viewBox="0 0 24 24"
              >
                <path
                  stroke="currentColor"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth="2"
                  d="M20 6L9 17l-5-5"
                ></path>
              </svg>
            )}
          </div>
          <div>
            <p className="text-sm font-medium">{title}</p>
            <p
              className="text-sm text-black-900/60"
              dangerouslySetInnerHTML={{ __html: message }}
            />
          </div>
        </div>
        <div className="w-12"></div>
        <Button style="ghost" onClick={dismissToast}>
          Dismiss
        </Button>
      </div>
    </>
  );
}

export default Toast;
