# typed: strict
# frozen_string_literal: true

class Forms::PostFormComponent < ApplicationComponent
  sig { params(form: PostForm).void }
  def initialize(form:)
    @form = form
  end
end
