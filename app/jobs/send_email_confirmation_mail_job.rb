# typed: strict
# frozen_string_literal: true

class SendEmailConfirmationMailJob < ApplicationJob
  sig { params(email_confirmation_id: T::Mewst::DatabaseId, locale: String).void }
  def perform(email_confirmation_id:, locale:)
    tasks = Mewst::CloudTasks.new
    tasks.create_task(
      path: "/internal/tasks/send_email_confirmation_mail",
      payload: {
        email_confirmation_id:,
        locale:
      }.to_json
    )
  end
end
