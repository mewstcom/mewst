# typed: true
# frozen_string_literal: true

class Internal::EmailConfirmations::Challenges::CreateController < Internal::ApplicationController
  include Latest::FormErrorable

  def call
    form = Internal::EmailConfirmationChallengeForm.new(
      email_confirmation_id: params[:email_confirmation_id],
      confirmation_code: params[:confirmation_code]
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::FormErrorResource, errors: form.errors)
    end

    ConfirmEmailUseCase.new.call(email_confirmation: form.email_confirmation.not_nil!)

    head :no_content
  end
end
