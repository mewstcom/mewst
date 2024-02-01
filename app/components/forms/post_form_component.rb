# typed: strict
# frozen_string_literal: true

class Forms::PostFormComponent < ApplicationComponent
  sig { params(from_modal: T::Boolean).void }
  def initialize(from_modal: false)
    @from_modal = from_modal
  end

  sig { returns(T::Boolean) }
  attr_reader :from_modal
  private :from_modal

  sig { returns(Integer) }
  private def textarea_tabindex
    from_modal ? 3 : 1
  end

  sig { returns(Integer) }
  private def submit_button_tabindex
    from_modal ? 4 : 2
  end

  sig { returns(Integer) }
  private def cancel_button_tabindex
    from_modal ? 5 : 0
  end
end
