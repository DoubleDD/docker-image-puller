import { CommonModule } from '@angular/common';
import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormsModule } from '@angular/forms';

@Component({
  selector: 'app-dropdown-selector',
  imports: [CommonModule, FormsModule],
  templateUrl: './dropdown-selector.component.html',
  styleUrl: './dropdown-selector.component.scss',
})
export class DropdownSelectorComponent {
  @Input() label: string = '';
  @Input() options: string[] = [];
  @Input() defaultOption: string = '';
  @Output() optionChange = new EventEmitter<string>();

  selectedOption: string = '';

  ngOnInit(): void {
    if (this.options.length > 0) {
      this.selectedOption = this.defaultOption || this.options[0]; // 默认选中第一个选项
    }
  }

  onOptionChange(event: Event): void {
    const target = event.target as HTMLSelectElement;
    this.selectedOption = target.value;
    this.optionChange.emit(this.selectedOption);
  }

  getValue(): string {
    return this.selectedOption;
  }
}
