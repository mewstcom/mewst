# typed: strict
# frozen_string_literal: true

class Forms::PostFormComponent < ApplicationComponent
  sig { params(form: PostForm, from_modal: T::Boolean).void }
  def initialize(form:, from_modal: false)
    @form = form
    @from_modal = from_modal
  end

  sig { returns(PostForm) }
  attr_reader :form
  private :form

  sig { returns(T::Boolean) }
  attr_reader :from_modal
  private :from_modal
end
