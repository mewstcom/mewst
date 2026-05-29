# typed: strict
# frozen_string_literal: true

class Modals::PostFormModalComponent < ApplicationComponent
  sig { params(form: PostForm).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(PostForm) }
  attr_reader :form
  private :form
end
