# typed: true
# frozen_string_literal: true

class Latest::CommentedPosts::CreateController < Latest::ApplicationController
  def call
    form = Forms::CommentedPost.new(
      comment: params[:comment]
    )
    form.profile = current_user!.profiles.find_by!(atname: params[:atname])

    if form.invalid?
      return render(
        json: Resources::Latest::ActiveModelErrors.new(form.errors),
        status: :unprocessable_entity
      )
    end

    result = ActiveRecord::Base.transaction do
      Services::CreateCommentedPost.new(form:).call
    end

    render(
      json: Resources::Latest::Post.new(result.post),
      status: :created
    )
  end
end
