import "solid-js";
import {
    BoxProps,
    TextProps,
    SpanProps,
    InputProps,
    SelectProps,
    AsciiFontProps,
    TabSelectProps,
    ScrollBoxProps,
    CodeProps,
    TextareaProps
} from "@opentui/solid";

declare module "solid-js" {
    namespace JSX {
        interface IntrinsicElements {
            box: BoxProps;
            text: TextProps;
            span: SpanProps;
            input: InputProps;
            select: SelectProps;
            ascii_font: AsciiFontProps;
            tab_select: TabSelectProps;
            scrollbox: ScrollBoxProps;
            code: CodeProps;
            textarea: TextareaProps;

            b: SpanProps;
            strong: SpanProps;
            i: SpanProps;
            em: SpanProps;
            u: SpanProps;
            br: {};
        }
    }
}
