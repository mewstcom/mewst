# typed: strict
# frozen_string_literal: true

class Forms::KeywordSearchFormComponent < ApplicationComponent
  sig { params(form: KeywordSearchForm, form_url: String).void }
  def initialize(form:, form_url:)
    @form = form
    @form_url = form_url
  end

  sig { returns(KeywordSearchForm) }
  attr_reader :form
  private :form

  sig { returns(String) }
  attr_reader :form_url
  private :form_url
end
