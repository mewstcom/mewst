# typed: true
# frozen_string_literal: true

class Internal::SignUp::CreateController < Internal::ApplicationController
  def call
    form = Forms::SignUp.new(
      atname: params[:atname],
      email: params[:email],
      locale: I18n.locale,
      password: params[:password]
    )

    if form.invalid?
      return render(
        json: Resources::Internal::ActiveModelErrors.new(form.errors),
        status: :unprocessable_entity
      )
    end

    result = Commands::SignUp.new(form:).call
    result.oauth_access_token.user.track_sign_in

    render(
      json: Resources::Internal::SignUp.new(result),
      status: :created
    )
  end
end
