import { Injectable } from '@nestjs/common';
import { PrismaService } from '../../prisma.service';

// Dependency injection — no `new PrismaService()` anywhere (Playbook: Ch. 3, DI)
@Injectable()
export class UsersRepository {
  constructor(private readonly db: PrismaService) {}

  async findByEmail(email: string) {
    return this.db.user.findUnique({ where: { email } });
  }

  async findById(id: string) {
    return this.db.user.findUnique({ where: { id } });
  }

  async create(data: { name: string; email: string; passwordHash: string }) {
    return this.db.user.create({ data });
  }

  async updateById(id: string, data: Partial<{ name: string; avatarUrl: string }>) {
    return this.db.user.update({ where: { id }, data });
  }
}
