import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { FormFilters } from "../../../v210/FormFilters";

it("uses server status counts and preserves the tag when changing status", () => {
 const onChange=vi.fn(); render(<FormFilters query={{status:"ACTIVE",tag:"third-party",limit:20}} facets={{status:{DRAFT:3,PENDING_APPROVAL:2,ACTIVE:5,PAUSED:1}}} onChange={onChange}/>);
 expect(screen.getByRole("button",{name:"All 11"})).toBeTruthy(); expect(screen.getByRole("button",{name:"Draft 3"})).toBeTruthy(); expect(screen.getByRole("button",{name:"Awaiting approval 2"})).toBeTruthy(); expect(screen.queryByRole("button",{name:/Rejected/})).toBeNull();
 fireEvent.click(screen.getByRole("button",{name:"Draft 3"})); expect(onChange).toHaveBeenCalledWith({status:"DRAFT",tag:"third-party",limit:20,cursor:undefined});
});
it("shows unknown counts without fabricating zeros or totals", () => {
 const onChange=vi.fn(); render(<FormFilters query={{limit:20}} onChange={onChange}/>); expect(screen.getByRole("button",{name:"All"})).toBeTruthy(); expect(screen.getByRole("button",{name:"Draft"})).toBeTruthy(); expect(screen.queryByRole("button",{name:"All 0"})).toBeNull();
});
it("applies name and tag without dropping the selected status", () => {
 const onChange=vi.fn(); render(<FormFilters query={{status:"ACTIVE",limit:20,cursor:"old"}} onChange={onChange}/>); fireEvent.change(screen.getByLabelText("Find a form"),{target:{value:" access "}}); fireEvent.change(screen.getByLabelText("Tag"),{target:{value:" vendor "}}); fireEvent.click(screen.getByRole("button",{name:"Search"})); expect(onChange).toHaveBeenCalledWith({status:"ACTIVE",limit:20,cursor:undefined,search:"access",tag:"vendor"});
});
it("withholds malformed or unrecognized facet counts", () => {
 const onChange=vi.fn(); const view=render(<FormFilters query={{limit:20}} facets={{status:{DRAFT:-1,ACTIVE:5}}} onChange={onChange}/>); expect(screen.getByRole("button",{name:"All"})).toBeTruthy(); expect(screen.queryByRole("button",{name:/All 4/})).toBeNull();
 view.rerender(<FormFilters query={{limit:20}} facets={{status:{DRAFT:1,UNKNOWN:9} as any}} onChange={onChange}/>); expect(screen.getByRole("button",{name:"All"})).toBeTruthy(); expect(screen.queryByRole("button",{name:/All 10/})).toBeNull();
});
