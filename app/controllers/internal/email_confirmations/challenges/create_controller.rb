# typed: true
# frozen_string_literal: true

class Internal::EmailConfirmations::Challenges::CreateController < Internal::ApplicationController
  def call
    form = Forms::EmailConfirmationChallenge.new(
      email_confirmation_id: params[:email_confirmation_id],
      confirmation_code: params[:confirmation_code]
    )

    if form.invalid?
      return render(
        json: Resources::Internal::ActiveModelErrors.new(form.errors),
        status: :unprocessable_entity
      )
    end

    Commands::ConfirmEmail.new(form:).call

    head :no_content
  end
end
