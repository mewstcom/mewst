# typed: true
# frozen_string_literal: true

class Latest::Reposts::CreateController < Latest::ApplicationController
  def call
    form = Forms::Repost.new(
      profile: current_profile!,
      target_post_id: params[:post_id]
    )

    if form.invalid?
      resource_errors = Latest::Entities::RepostFormError.build_from_errors(errors: form.errors)

      return render(
        json: Latest::Resources::ResponseError.new(resource_errors),
        status: :unprocessable_entity
      )
    end

    result = ActiveRecord::Base.transaction do
      Services::CreateRepost.new(form:).call
    end

    render(
      json: Latest::Presenters::Repost.new(post: result.post),
      status: :created
    )
  end
end
