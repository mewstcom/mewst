# typed: true
# frozen_string_literal: true

class Internal::Sessions::CreateController < Internal::ApplicationController
  def call
    form = Forms::Session.new(
      email: params[:email],
      password: params[:password]
    )

    if form.invalid?
      return render(
        json: Resources::Internal::ActiveModelErrors.new(form.errors),
        status: :unprocessable_entity
      )
    end

    result = ApplicationRecord.transaction do
      Services::CreateSession.new(form:).call
    end

    render(
      json: Resources::Internal::Account.new(result),
      status: :created
    )
  end
end
