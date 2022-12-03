# typed: strict
# frozen_string_literal: true

class Inputs::PhoneNumberInputComponent < ApplicationComponent
  sig { params(form: ActionView::Helpers::FormBuilder, field_name: Symbol, autofocus: T::Boolean).void }
  def initialize(form:, field_name:, autofocus: true)
    @form = form
    @field_name = field_name
    @autofocus = autofocus
  end
end
