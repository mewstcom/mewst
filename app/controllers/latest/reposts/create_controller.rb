# typed: true
# frozen_string_literal: true

class Latest::Reposts::CreateController < Latest::ApplicationController
  def call
    form = Latest::RepostForm.new(
      viewer: current_profile!,
      target_post_id: params[:post_id]
    )

    if form.invalid?
      return response_form_errors(resource_class: Latest::RepostFormErrorResource, errors: form.errors)
    end

    ActiveRecord::Base.transaction do
      input = CreateRepostService::Input.from_latest_form(form:)
      CreateRepostService.new.call(input:)
    end

    head :no_content
  end
end
