# typed: strict
# frozen_string_literal: true

class Inputs::PhoneNumberInputComponent < ApplicationComponent
  sig { params(form: ActionView::Helpers::FormBuilder, field_name: Symbol).void }
  def initialize(form:, field_name:)
    @form = form
    @field_name = field_name
  end
end
