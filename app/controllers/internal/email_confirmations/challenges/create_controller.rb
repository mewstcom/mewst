# typed: true
# frozen_string_literal: true

class Internal::EmailConfirmations::Challenges::CreateController < Internal::ApplicationController
  def call
    form = Internal::EmailConfirmationChallengeForm.new(
      email_confirmation_id: params[:email_confirmation_id],
      confirmation_code: params[:confirmation_code]
    )

    if form.invalid?
      resources = Internal::FormErrorResource.build_from_errors(errors: form.errors)
      return render(
        json: Internal::ResponseErrorSerializer.new(resources),
        status: :unprocessable_entity
      )
    end

    ConfirmEmailUseCase.new.call(email_confirmation: form.email_confirmation.not_nil!)

    head :no_content
  end
end
