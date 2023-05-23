# typed: strict
# frozen_string_literal: true

class SendVerificationMailJob < ApplicationJob
  sig { params(verification_id: String, locale: String).void }
  def perform(verification_id:, locale:)
    tasks = Mewst::CloudTasks.new
    tasks.create_task(
      path: "/api/internal/tasks/send_verification_mail",
      payload: {
        verification_id:,
        locale:
      }.to_json
    )
  end
end
