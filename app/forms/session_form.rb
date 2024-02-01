# typed: strict
# frozen_string_literal: true

class V1::SessionForm < V1::ApplicationForm
  attribute :email, :string
  attribute :password, :string

  validates :email, email: true, presence: true
  validates :password, presence: true
  validates :user, presence: true
  validate :authentication

  sig { returns(T.nilable(User)) }
  def user
    @user ||= T.let(User.find_by(email:), T.nilable(User))
  end

  sig { void }
  private def authentication
    unless user.not_nil!.authenticate(password)
      errors.add(:base, :unauthenticated)
    end
  end
end
