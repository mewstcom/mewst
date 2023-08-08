# typed: strict
# frozen_string_literal: true

class ConfirmEmailService < ApplicationService
  class Result < T::Struct; end

  sig { params(form: Forms::EmailConfirmationChallenge).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Result) }
  def call
    form.email_confirmation!.success

    Result.new
  end

  sig { returns(Forms::EmailConfirmationChallenge) }
  attr_reader :form
  private :form
end
