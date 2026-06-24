// Type declarations for Wails v3 runtime modules
// See: https://v3.wails.io/reference/frontend-runtime

declare module '@wailsio/runtime' {
  export namespace Call {
    /**
     * Calls a bound Go service method by its fully-qualified name
     * in the format 'package.struct.method'.
     * @returns The return value(s) from the Go method.
     *          Multi-value returns come back as an array.
     *          If the Go method's last return is a non-nil error,
     *          the promise rejects with a RuntimeError.
     */
    export function ByName(methodName: string, ...args: any[]): Promise<any>

    /**
     * Calls a bound Go service method by its numeric (uint32 hash) ID.
     */
    export function ByID(methodID: number, ...args: any[]): Promise<any>

    /**
     * Low-level call with a CallOptions descriptor.
     */
    export function Call(options: {
      methodName?: string
      methodID?: number
      args?: any[]
    }): Promise<any>
  }

  export namespace Events {
    export function On(event: string, callback: (event: { data: any }) => void): () => void
    export function Emit(event: string, data?: any): Promise<boolean>
  }

  export namespace System {
    export function invoke(message: string): void
  }

  export namespace Window {
    export function SetTitle(title: string): Promise<void>
    export function Center(): Promise<void>
    export function Minimise(): Promise<void>
    export function Maximise(): Promise<void>
    export function Unmaximise(): Promise<void>
    export function SetFullscreen(fullscreen: boolean): Promise<void>
    export function Close(): Promise<void>
  }

  export namespace Clipboard {
    export function SetText(text: string): Promise<void>
    export function Text(): Promise<string>
  }

  export namespace Dialogs {
    export function Question(options: {
      Title: string
      Message: string
      Buttons: Array<{ Label: string; IsDefault?: boolean }>
    }): Promise<{ Response: number }>
  }

  export { Call as default }
}
